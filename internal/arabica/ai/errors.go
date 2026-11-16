package ai

import (
	"errors"
	"sort"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

// ProviderFailure is safe to log: all text is application-defined, never raw
// upstream errors, model content, credentials, or user/library values.
type ProviderFailure struct {
	Stage           string
	Reason          string
	Transport       provider.FailureDetails
	FinishReason    string
	MessageRole     string
	ContentBytes    int
	ToolCalls       int
	Field           string
	ValidationIssue string
	UnknownFields   int
}

func (f *ProviderFailure) Error() string {
	return "assistant provider failure: " + f.Stage + "/" + f.Reason
}
func (f *ProviderFailure) Unwrap() error { return ErrProvider }
func providerRequestFailure(err error) error {
	if errors.Is(err, provider.ErrRequestTooLarge) {
		return ErrProviderRequestTooLarge
	}
	details := provider.DescribeFailure(err)
	return &ProviderFailure{
		Stage: "request", Reason: "upstream_failure", Transport: details,
		FinishReason: details.FinishReason, MessageRole: details.MessageRole,
		ContentBytes: details.ContentBytes, ToolCalls: details.ToolCalls,
	}
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
	rules, ok := fields[kind]
	if !ok {
		f.ValidationIssue = "invalid_kind"
		return f
	}
	names := make([]string, 0, len(patch))
	for name := range patch {
		names = append(names, name)
		if _, ok := rules[name]; !ok {
			f.UnknownFields++
		}
	}
	sort.Strings(names)
	if f.UnknownFields > 0 {
		f.ValidationIssue = "unknown_fields"
		for _, name := range names {
			if _, ok := rules[name]; !ok && diagnosticFieldName(name) {
				f.Field = name
				break
			}
		}
		return f
	}
	f.ValidationIssue = "invalid_value"
	for _, name := range names {
		if validateFields(kind, map[string]any{name: patch[name]}) != nil {
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

func diagnosticFieldName(name string) bool {
	for _, rules := range fields {
		if _, ok := rules[name]; ok {
			return true
		}
	}
	switch name {
	case "in_scope", "reply", "kind", "patch", "choices", "recent_brew_id", "recent_brews", "espresso_params", "pourover_params":
		return true
	default:
		return false
	}
}
