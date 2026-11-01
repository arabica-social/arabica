package ai

import "encoding/json"

// ContentPart is an OpenAI-compatible text or inline-image content part.
// Callers own validation and must never supply untrusted remote image URLs.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}
type ImageURL struct {
	URL string `json:"url"`
}

// MarshalJSON keeps the historical string content encoding for text-only calls.
func (m Message) MarshalJSON() ([]byte, error) {
	type plain Message
	if len(m.Parts) == 0 {
		return json.Marshal(plain(m))
	}
	return json.Marshal(struct {
		Role       string        `json:"role"`
		Content    []ContentPart `json:"content"`
		ToolCallID string        `json:"tool_call_id,omitempty"`
		ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	}{m.Role, m.Parts, m.ToolCallID, m.ToolCalls})
}
