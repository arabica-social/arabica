package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message supports text conversation and tool-call/result round trips.
type Message struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	Parts      []ContentPart `json:"-"`
}

// Tool deliberately leaves provider-specific schema and strictness to callers.
type Tool struct {
	Type     string          `json:"type"`
	Function json.RawMessage `json:"function"`
}
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}
type Completion struct {
	Message      Message
	FinishReason string
	Usage        *Usage
}

// Client is immutable after construction and safe for concurrent requests.
// It never logs request bodies, credentials, URLs or upstream error bodies.
type Client struct {
	provider     ProviderConfig
	key          string
	http         *http.Client
	providerName string
	observe      func(Observation)
}

// NewClient resolves APIKey interpolation once. An omitted/empty key permits
// intentionally unauthenticated providers; unresolved variables fail startup.
func NewClient(config Config) (*Client, error) {
	return NewClientWithObserver(config, nil)
}

// NewClientWithObserver reports upstream calls synchronously. The observer must
// be concurrency-safe and must not block; it never receives request content.
func NewClientWithObserver(config Config, observe func(Observation)) (*Client, error) {
	c, err := normalize(config)
	if err != nil {
		return nil, err
	}
	p := c.Providers[c.DefaultProvider]
	key, err := resolveAPIKey(p.APIKey)
	if err != nil {
		return nil, err
	}
	p.APIKey = "" // Retain the resolved secret only in the dedicated private field.
	return &Client{provider: p, key: key, providerName: c.DefaultProvider, observe: observe, http: &http.Client{
		Timeout:       time.Duration(p.TimeoutSeconds) * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

// Complete makes exactly one non-streaming request. Tools and response format
// are caller-owned encodings, not authorization or schema validation policies.
func (c *Client) Complete(ctx context.Context, messages []Message, tools []Tool, responseFormat json.RawMessage) (completion Completion, err error) {
	var zero Completion
	if len(messages) == 0 {
		return zero, errors.New("ai: messages required")
	}
	request := struct {
		Model          string          `json:"model"`
		Messages       []Message       `json:"messages"`
		Tools          []Tool          `json:"tools,omitempty"`
		ResponseFormat json.RawMessage `json:"response_format,omitempty"`
		MaxTokens      int             `json:"max_tokens"`
		Stream         bool            `json:"stream"`
	}{c.provider.Model, messages, tools, responseFormat, c.provider.MaxTokens, false}
	body, err := json.Marshal(request)
	if err != nil {
		return zero, errors.New("ai: invalid request JSON")
	}
	if int64(len(body)) > c.provider.MaxRequestBytes {
		return zero, ErrRequestTooLarge
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.provider.TimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.provider.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return zero, errors.New("ai: cannot construct request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	observation := Observation{Provider: c.providerName, Model: c.provider.Model, UsageStatus: "unavailable"}
	started := time.Now()
	defer func() {
		if c.observe != nil {
			observation.Duration = time.Since(started)
			if err != nil {
				observation.Failure = DescribeFailure(err)
			}
			c.observe(observation)
		}
	}()
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, networkFailure(err)
	}
	defer resp.Body.Close()
	observation.HTTPStatus = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return zero, upstreamHTTPFailure(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, c.provider.MaxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		return zero, failed("response_read", "ai: cannot read provider response", 0)
	}
	if int64(len(data)) > c.provider.MaxResponseBytes {
		return zero, failed("response_too_large", "ai: provider response exceeds limit", 0)
	}
	var result struct {
		Usage   json.RawMessage `json:"usage"`
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil {
		return zero, failed("invalid_provider_response", "ai: invalid provider response", 0)
	}
	observation.Usage, observation.UsageStatus = parseUsage(result.Usage)
	if len(result.Choices) != 1 {
		return zero, failed("invalid_provider_response", "ai: invalid provider response", 0)
	}
	choice := result.Choices[0]
	if choice.Message.Role != "assistant" || (choice.Message.Content == "" && len(choice.Message.ToolCalls) == 0) {
		return zero, failed("invalid_assistant_message", "ai: invalid assistant message", 0)
	}
	return Completion{Message: choice.Message, FinishReason: choice.FinishReason, Usage: observation.Usage}, nil
}
