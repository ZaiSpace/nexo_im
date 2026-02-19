package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

const (
	TraceIDHeader       = "Trace-Id"
	XTraceIDHeader      = "X-Trace-Id"
	TraceIDContextKey   = "trace_id"
	operationIDQueryKey = "operation_id"
)

// TraceID injects trace_id into context and echoes it in response header.
// It also writes the trace header back to request headers so adaptor-based
// handlers (e.g. websocket net/http handlers) can read the same value.
func TraceID() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		traceID := resolveTraceID(ctx, c)
		ctx = WithTraceID(ctx, traceID)

		c.Set(TraceIDContextKey, traceID)
		c.Request.Header.Set(TraceIDHeader, traceID)
		c.Request.Header.Set(XTraceIDHeader, traceID)
		c.Response.Header.Set(TraceIDHeader, traceID)
		c.Response.Header.Set(XTraceIDHeader, traceID)

		c.Next(ctx)

		c.Response.Header.Set(TraceIDHeader, traceID)
		c.Response.Header.Set(XTraceIDHeader, traceID)
	}
}

// GetTraceID returns trace ID from context.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(TraceIDContextKey); v != nil {
		if traceID, ok := v.(string); ok {
			return strings.TrimSpace(traceID)
		}
	}
	return ""
}

// WithTraceID returns a new context carrying trace ID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TraceIDContextKey, traceID)
}

func resolveTraceID(ctx context.Context, c *app.RequestContext) string {
	traceID := strings.TrimSpace(string(c.GetHeader(TraceIDHeader)))
	if traceID == "" {
		traceID = strings.TrimSpace(string(c.GetHeader(XTraceIDHeader)))
	}
	if traceID == "" {
		traceID = strings.TrimSpace(c.Query(operationIDQueryKey))
	}
	if traceID == "" {
		traceID = GetTraceID(ctx)
	}
	if traceID == "" {
		traceID = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return traceID
}
