package ai

import (
	"errors"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

// ProviderFailure is safe to log: all text is application-defined, never raw
// upstream errors, model content, credentials, or user/library values.
type ProviderFailure struct {
	Stage        string
	Reason       string
	Transport    provider.FailureDetails
	FinishReason string
	ContentBytes int
	ToolCalls    int
	Field        string
}

func (f *ProviderFailure) Error() string {
	return "assistant provider failure: " + f.Stage + "/" + f.Reason
}
func (f *ProviderFailure) Unwrap() error { return ErrProvider }
func providerRequestFailure(err error) error {
	if errors.Is(err, provider.ErrRequestTooLarge) {
		return ErrProviderRequestTooLarge
	}
	return &ProviderFailure{Stage: "request", Reason: "upstream_failure", Transport: provider.DescribeFailure(err)}
}
func rejected(reason string, c provider.Completion) *ProviderFailure {
	finish := "other"
	switch c.FinishReason {
	case "stop", "length", "content_filter", "tool_calls", "function_call", "":
		finish = c.FinishReason
	}
	return &ProviderFailure{Stage: "response", Reason: reason, FinishReason: finish, ContentBytes: len(c.Message.Content), ToolCalls: len(c.Message.ToolCalls)}
}
func rejectedFields(kind string, patch map[string]any, c provider.Completion) error {
	f := rejected("invalid_draft_fields", c)
	// Only allowlisted field names can enter logs; unknown model keys are withheld.
	for name := range fields[kind] {
		if value, ok := patch[name]; ok && validateFields(kind, map[string]any{name: value}) != nil {
			f.Field = name
			break
		}
	}
	return f
}

// ProviderDiagnostic extracts the safe failure, without ever formatting an
// arbitrary Completer error (implementations can include credentials in errors).
func ProviderDiagnostic(err error) *ProviderFailure {
	var f *ProviderFailure
	if errors.As(err, &f) {
		return f
	}
	return &ProviderFailure{Stage: "unknown", Reason: "provider_failure"}
}
