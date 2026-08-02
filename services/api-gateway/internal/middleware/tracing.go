package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "api-gateway"

// TracingMiddleware injects an OpenTelemetry span for each incoming request and
// propagates W3C trace-context headers to upstream services.
func TracingMiddleware() fiber.Handler {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		// Extract parent context from incoming headers (W3C traceparent / tracestate).
		carrier := propagation.MapCarrier{}
		c.Request().Header.VisitAll(func(k, v []byte) {
			carrier[string(k)] = string(v)
		})
		ctx := propagator.Extract(c.UserContext(), carrier)

		// Ensure a unique Request-ID exists.
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
			c.Set("X-Request-ID", reqID)
		}

		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(c.Method()),
				semconv.HTTPURL(c.OriginalURL()),
				semconv.HTTPRoute(c.Route().Path),
				attribute.String("http.request_id", reqID),
				attribute.String("net.peer.ip", c.IP()),
			),
		)
		defer span.End()

		// Inject propagation headers so upstream services can continue the trace.
		outCarrier := propagation.MapCarrier{}
		propagator.Inject(ctx, outCarrier)
		for k, v := range outCarrier {
			c.Request().Header.Set(k, v)
		}

		c.SetUserContext(ctx)

		err := c.Next()

		span.SetAttributes(semconv.HTTPStatusCode(c.Response().StatusCode()))
		if err != nil {
			span.RecordError(err)
		}
		return err
	}
}
