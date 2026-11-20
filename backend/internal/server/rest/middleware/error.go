package restmiddleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	authusecase "github.com/azhkadaes/vms/backend/internal/usecase/auth"
	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

type apiError struct {
	Code        string              `json:"code"`
	Message     string              `json:"message"`
	RequestID   string              `json:"requestId"`
	FieldErrors map[string][]string `json:"fieldErrors,omitempty"`
}

// statusCodes is the catalogue code used when only a status is known.
var statusCodes = map[int]string{
	400: "invalid_request",
	401: "unauthenticated",
	403: "forbidden",
	404: "not_found",
	405: "method_not_allowed",
	409: "invalid_state",
	413: "payload_too_large",
	415: "unsupported_media_type",
	422: "validation_failed",
	429: "rate_limited",
	500: "internal_error",
	503: "service_unavailable",
}

const workspaceMessage = "Selesaikan pemilihan peran atau perubahan kata sandi."

type outcome struct {
	status     int
	code       string
	message    string
	fields     map[string][]string
	retryAfter int
}

// Failure writes {code, message, requestId, fieldErrors?} with the catalogue message for
// the code. Causes never reach the body; 5xx log only a redacted cause class.
func Failure(c fiber.Ctx, err error, logger *slog.Logger) error {
	out := classify(err)
	if out.retryAfter > 0 {
		c.Set("Retry-After", strconv.Itoa(out.retryAfter))
	}
	id, _ := c.Locals("vms.requestId").(string)
	if id == "" {
		generated, randomErr := uuid.NewRandom()
		if randomErr == nil {
			id = generated.String()
		} else {
			id = "unavailable"
		}
	}
	c.Set("X-Request-ID", id)
	if out.status >= 500 {
		attrs := []any{"request_id", id, "status", out.status, "code", out.code, "method", c.Method(), "route", c.Route().Path, "cause", causeClass(err)}
		if state := sqlState(err); state != "" {
			attrs = append(attrs, "sqlstate", state)
		}
		logger.ErrorContext(c.Context(), "VMS request failed", attrs...)
	}
	c.Set("Cache-Control", "no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Status(out.status).JSON(apiError{Code: out.code, Message: out.message, RequestID: id, FieldErrors: out.fields})
}

// classify maps an error to a status and catalogue code. Explicit failures win, then
// auth sentinels, repository not-found, Fiber transport errors and deadlines.
func classify(err error) outcome {
	var business *failure.Error
	var protocol *fiber.Error
	switch {
	case errors.As(err, &business):
		out := forStatus(business.Status, business.Code)
		if out.code == business.Code && out.code == "validation_failed" {
			out.fields = nonEmpty(business.Fields)
		}
		out.retryAfter = business.RetryAfter
		return out
	case errors.Is(err, authusecase.ErrUnauthenticated):
		return forStatus(401, "unauthenticated")
	case errors.Is(err, authusecase.ErrRoleChanged):
		return forStatus(409, "role_changed")
	case errors.Is(err, authusecase.ErrWorkspaceBlocked):
		out := forStatus(403, "forbidden")
		out.message = workspaceMessage
		return out
	case errors.Is(err, failure.ErrNotFound):
		return forStatus(404, "not_found")
	case errors.As(err, &protocol):
		return forStatus(protocol.Code, protocol.Message)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return forStatus(503, "service_unavailable")
	}
	return forStatus(500, "internal_error")
}

// nonEmpty drops blank messages and keys left without any; nil when nothing remains.
func nonEmpty(fields map[string][]string) map[string][]string {
	var out map[string][]string
	for key, messages := range fields {
		for _, message := range messages {
			if message == "" {
				continue
			}
			if out == nil {
				out = map[string][]string{}
			}
			out[key] = append(out[key], message)
		}
	}
	return out
}

// forStatus keeps a catalogue code; anything else falls back to the status default.
// Statuses outside the catalogue become 400 or 500 with that status's default code.
func forStatus(status int, code string) outcome {
	if _, known := statusCodes[status]; !known {
		if status >= 400 && status < 500 {
			status = 400
		} else {
			status = 500
		}
		code = ""
	}
	message, known := failure.Message(code)
	if !known {
		code = statusCodes[status]
		message, _ = failure.Message(code)
	}
	return outcome{status: status, code: code, message: message}
}

// causeClass names the innermost error type without its text, so operators can triage
// without logging request bodies, SQL, tokens, credentials or personal data.
func causeClass(err error) string {
	if err == nil {
		return "none"
	}
	if errors.Is(err, context.Canceled) {
		return "context.Canceled"
	}
	leaf := err
	for depth := 0; depth < 64; depth++ {
		next := unwrapOne(leaf)
		if next == nil {
			break
		}
		leaf = next
	}
	return strings.TrimPrefix(fmt.Sprintf("%T", leaf), "*")
}

func unwrapOne(err error) error {
	switch e := err.(type) {
	case interface{ Unwrap() error }:
		return e.Unwrap()
	case interface{ Unwrap() []error }:
		if causes := e.Unwrap(); len(causes) > 0 {
			return causes[0]
		}
	}
	return nil
}

// sqlState returns a driver SQLSTATE (five upper-case alphanumerics) found in the chain.
func sqlState(err error) string {
	var coded interface{ SQLState() string }
	if !errors.As(err, &coded) {
		return ""
	}
	state := coded.SQLState()
	if len(state) != 5 {
		return ""
	}
	for _, r := range state {
		if (r < '0' || r > '9') && (r < 'A' || r > 'Z') {
			return ""
		}
	}
	return state
}
