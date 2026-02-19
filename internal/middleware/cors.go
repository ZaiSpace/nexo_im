package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// CORS is the CORS middleware
func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Token, Ignore-Auth, X-Service-Name, X-Timestamp, X-Signature, X-User-Id, X-Platform-Id, Trace-Id, X-Trace-Id")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Trace-Id, X-Trace-Id")
		c.Header("Access-Control-Allow-Credentials", "true")

		if string(c.Method()) == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next(ctx)
	}
}
