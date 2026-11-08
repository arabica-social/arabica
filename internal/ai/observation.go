package ai

import (
	"encoding/json"
	"time"
)

// Usage contains provider-reported counts, not estimates. Cached input is a
// subset of input; reasoning tokens are already included in output.
type Usage struct {
	InputTokens       int64
	OutputTokens      int64
	CachedInputTokens int64
}

// Observation describes one attempted upstream call, including failed calls.
// Provider and Model come from server configuration, never from responses.
type Observation struct {
	Provider    string
	Model       string
	Duration    time.Duration
	HTTPStatus  int
	Failure     FailureDetails
	Usage       *Usage
	UsageStatus string
}

func parseUsage(raw json.RawMessage) (*Usage, string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, "missing"
	}
	var wire struct {
		Input   *int64 `json:"prompt_tokens"`
		Output  *int64 `json:"completion_tokens"`
		Details struct {
			Cached int64 `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Input == nil || wire.Output == nil {
		return nil, "invalid"
	}
	if *wire.Input < 0 || *wire.Output < 0 || wire.Details.Cached < 0 || wire.Details.Cached > *wire.Input {
		return nil, "invalid"
	}
	return &Usage{InputTokens: *wire.Input, OutputTokens: *wire.Output, CachedInputTokens: wire.Details.Cached}, "reported"
}
