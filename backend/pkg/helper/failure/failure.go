package failure

import "errors"

// ErrNotFound marks a missing or out-of-scope record. Repositories wrap it
// (fmt.Errorf("...: %w", ...)) and the HTTP error handler answers 404 not_found.
var ErrNotFound = errors.New("not_found")

// Error carries only allowlisted client information. Causes stay out of HTTP responses.
type Error struct {
	Status     int
	Code       string
	Fields     map[string][]string
	RetryAfter int
	cause      error
}

func (e *Error) Error() string { return e.Code }

// Unwrap exposes the cause to errors.Is/As and the redacted 5xx log line only.
func (e *Error) Unwrap() error { return e.cause }

// WithCause keeps the underlying error for operators; it never reaches the response body.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// New builds a status/code failure; the code must be in the message catalogue.
func New(status int, code string) *Error { return &Error{Status: status, Code: code} }

// ValidationBuilder collects 422 field errors keyed by the flat form key ("visitors", not "visitors.3").
type ValidationBuilder struct {
	fields map[string][]string
}

// Validation starts an empty field-error collection.
func Validation() *ValidationBuilder { return &ValidationBuilder{fields: map[string][]string{}} }

// Add records a user-facing Indonesian message for key; repeated messages are kept once.
func (b *ValidationBuilder) Add(key, message string) *ValidationBuilder {
	if message == "" {
		message = defaultFieldMessage
	}
	for _, existing := range b.fields[key] {
		if existing == message {
			return b
		}
	}
	b.fields[key] = append(b.fields[key], message)
	return b
}

// Has reports whether key already has a message, so later checks can skip it.
func (b *ValidationBuilder) Has(key string) bool { return len(b.fields[key]) > 0 }

// Err returns nil when nothing was added, otherwise a detached 422 validation_failed failure.
func (b *ValidationBuilder) Err() error {
	if len(b.fields) == 0 {
		return nil
	}
	fields := make(map[string][]string, len(b.fields))
	for key, messages := range b.fields {
		fields[key] = append([]string(nil), messages...)
	}
	return &Error{Status: 422, Code: "validation_failed", Fields: fields}
}

const defaultFieldMessage = "Periksa nilai ini."
