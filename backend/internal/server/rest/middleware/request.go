package restmiddleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/azhkadaes/vms/backend/pkg/helper/failure"
)

// RequestContext sets correlation, response headers and a bounded standard context.
func RequestContext(timeout time.Duration, tracer trace.Tracer) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := uuid.NewRandom()
		if err != nil {
			message, _ := failure.Message("service_unavailable")
			return c.Status(503).JSON(apiError{Code: "service_unavailable", Message: message, RequestID: "unavailable"})
		}
		c.Locals("vms.requestId", id.String())
		c.Set("X-Request-ID", id.String())
		c.Set("Cache-Control", "no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		ctx, span := tracer.Start(ctx, "http.request", trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		c.SetContext(ctx)
		err = c.Next()
		span.SetAttributes(attribute.String("http.request.method", c.Method()), attribute.String("http.route", c.Route().Path))
		if err != nil {
			span.SetStatus(codes.Error, "request failed")
		}
		return err
	}
}
