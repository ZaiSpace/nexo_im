package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mbeoliero/kit/log"
)

const maxLogBodyBytes = 2048

// Logger logs request/response summary for each HTTP request.
func Logger() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		startAt := time.Now()
		clientIP := c.ClientIP()
		method := string(c.Method())
		uri := string(c.Path())
		isWS := isWebSocketHandshake(c)
		reqBody := formatBody(c.Request.Body(), isWS)

		c.Next(ctx)

		status := c.Response.StatusCode()
		respBody := formatBody(c.Response.Body(), status == http.StatusSwitchingProtocols)
		cost := time.Since(startAt)

		if status >= http.StatusBadRequest {
			log.CtxWarn(ctx, "[%s] %s %s status=%d cost=%s req=%s resp=%s", clientIP, method, uri, status, cost, reqBody, respBody)
			return
		}

		log.CtxInfo(ctx, "[%s] %s %s status=%d cost=%s req=%s resp=%s", clientIP, method, uri, status, cost, reqBody, respBody)
	}
}

func isWebSocketHandshake(c *app.RequestContext) bool {
	if !strings.EqualFold(string(c.Method()), http.MethodGet) {
		return false
	}

	upgrade := strings.TrimSpace(string(c.GetHeader("Upgrade")))
	if !strings.EqualFold(upgrade, "websocket") {
		return false
	}

	connection := strings.ToLower(strings.TrimSpace(string(c.GetHeader("Connection"))))
	if !strings.Contains(connection, "upgrade") {
		return false
	}

	return len(c.GetHeader("Sec-WebSocket-Key")) > 0
}

func formatBody(body []byte, skip bool) string {
	if skip || len(body) == 0 {
		return "-"
	}
	if len(body) <= maxLogBodyBytes {
		return string(body)
	}
	return fmt.Sprintf("%s...(truncated,total=%dB)", string(body[:maxLogBodyBytes]), len(body))
}
