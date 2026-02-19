package sdk

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/stretchr/testify/require"
)

func TestApplyAuthHeaders_WithTraceIDFromContext(t *testing.T) {
	c := &Client{}
	req := &protocol.Request{}
	ctx := context.WithValue(context.Background(), traceIDContextKey, "trace-from-ctx")

	c.applyAuthHeaders(ctx, req, "GET", "/health", nil, nil)

	require.Equal(t, "trace-from-ctx", string(req.Header.Peek(traceIDHeader)))
	require.Equal(t, "trace-from-ctx", string(req.Header.Peek(xTraceIDHeader)))
}

func TestApplyAuthHeaders_WithoutTraceIDFromContext(t *testing.T) {
	c := &Client{}
	req := &protocol.Request{}

	c.applyAuthHeaders(context.Background(), req, "GET", "/health", nil, nil)

	require.Empty(t, string(req.Header.Peek(traceIDHeader)))
	require.Empty(t, string(req.Header.Peek(xTraceIDHeader)))
}

func TestApplyAuthHeaders_WithTraceIDHeaderKeyInContext(t *testing.T) {
	c := &Client{}
	req := &protocol.Request{}
	ctx := context.WithValue(context.Background(), traceIDHeader, "trace-from-header-key")

	c.applyAuthHeaders(ctx, req, "GET", "/health", nil, nil)

	require.Equal(t, "trace-from-header-key", string(req.Header.Peek(traceIDHeader)))
	require.Equal(t, "trace-from-header-key", string(req.Header.Peek(xTraceIDHeader)))
}

func TestApplyAuthHeaders_WithTraceIDBytesInContext(t *testing.T) {
	c := &Client{}
	req := &protocol.Request{}
	ctx := context.WithValue(context.Background(), traceIDContextKey, []byte("trace-from-bytes"))

	c.applyAuthHeaders(ctx, req, "GET", "/health", nil, nil)

	require.Equal(t, "trace-from-bytes", string(req.Header.Peek(traceIDHeader)))
	require.Equal(t, "trace-from-bytes", string(req.Header.Peek(xTraceIDHeader)))
}
