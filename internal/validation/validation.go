// Package validation provides the ValidationError type used by entity
// validators to return per-field error messages. It lives in a separate
// package to avoid import cycles: entity model packages (e.g.
// arabica/entities) import this, while the handlers package depends on
// entities and provides the FieldError interface check.
package validation

import "errors"

// Error is a concrete field-validation error that collects per-field
// messages. Validators construct it with NewError and add field errors
// via AddErr. It implements the error interface and exposes FieldErrors()
// so handlers can emit the documented {error, code, fields} JSON envelope.
type Error struct {
	fields    map[string]string
	sentinels map[string]error
}

// New returns an empty Error ready to collect field errors.
func New() *Error {
	return &Error{
		fields:    make(map[string]string),
		sentinels: make(map[string]error),
	}
}

// Add records a validation message for the given field. If the field
// already has a message, the first one wins.
func (v *Error) Add(field, message string) *Error {
	if _, ok := v.fields[field]; !ok {
		v.fields[field] = message
	}
	return v
}

// AddErr records a sentinel error for the given field. The sentinel is
// preserved so errors.Is can match it, while the human-readable message
// is extracted for the fields map.
func (v *Error) AddErr(field string, err error) *Error {
	if _, ok := v.fields[field]; !ok {
		v.fields[field] = err.Error()
		v.sentinels[field] = err
	}
	return v
}

// HasErrors reports whether any field errors have been recorded.
func (v *Error) HasErrors() bool {
	return len(v.fields) > 0
}

// Error implements the error interface, returning the first field message
// for backwards compatibility with callers that display err.Error().
func (v *Error) Error() string {
	if len(v.fields) == 0 {
		return "validation failed"
	}
	for _, msg := range v.fields {
		return msg
	}
	return "validation failed"
}

// FieldErrors returns the per-field error map.
func (v *Error) FieldErrors() map[string]string {
	return v.fields
}

// Is implements errors.Is support so existing tests using errors.Is
// against sentinel errors (e.g. ErrNameRequired) still match.
func (v *Error) Is(target error) bool {
	for _, err := range v.sentinels {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
