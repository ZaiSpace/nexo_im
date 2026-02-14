package middleware

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/ZaiSpace/nexo_im/internal/config"
	"github.com/ZaiSpace/nexo_im/pkg/errcode"
	"github.com/ZaiSpace/nexo_im/pkg/jwt"
	"github.com/ZaiSpace/nexo_im/pkg/response"
)

const (
	// AuthorizationHeader is the header key for authorization
	AuthorizationHeader = "Authorization"
	// XTokenHeader is the fallback header key used by external systems
	XTokenHeader = "X-Token"
	// IgnoreAuthHeader can be used in TEST env to bypass auth
	IgnoreAuthHeader = "Ignore-Auth"
	// BearerPrefix is the prefix for bearer token
	BearerPrefix = "Bearer "
	// UserIdKey is the context key for user Id
	UserIdKey = "user_id"
	// PlatformIdKey is the context key for platform Id
	PlatformIdKey = "platform_id"
)

// JWTAuth is the JWT authentication middleware
func JWTAuth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		if isTestEnv() && len(c.GetHeader(IgnoreAuthHeader)) != 0 {
			c.Next(ctx)
			return
		}

		tokenString, err := extractToken(c)
		if errors.Is(err, errcode.ErrTokenMissing) {
			response.ErrorWithCode(ctx, c, errcode.ErrTokenMissing)
			c.Abort()
			return
		}
		if err != nil {
			response.ErrorWithCode(ctx, c, errcode.ErrTokenInvalid)
			c.Abort()
			return
		}

		claims, err := ParseTokenWithFallback(tokenString, config.GlobalConfig)
		if err != nil {
			response.ErrorWithCode(ctx, c, errcode.ErrTokenInvalid)
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(UserIdKey, claims.UserId)
		c.Set(PlatformIdKey, claims.PlatformId)

		c.Next(ctx)
	}
}

func extractToken(c *app.RequestContext) (string, error) {
	authHeader := strings.TrimSpace(string(c.GetHeader(AuthorizationHeader)))
	if authHeader != "" {
		if strings.HasPrefix(authHeader, BearerPrefix) {
			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, BearerPrefix))
			if tokenString == "" {
				return "", errcode.ErrTokenMissing
			}
			return tokenString, nil
		}
		// If Authorization is malformed, still allow X-Token as fallback for compatibility.
		xToken := strings.TrimSpace(string(c.GetHeader(XTokenHeader)))
		if xToken != "" {
			return xToken, nil
		}
		return "", errcode.ErrTokenInvalid
	}

	xToken := strings.TrimSpace(string(c.GetHeader(XTokenHeader)))
	if xToken == "" {
		return "", errcode.ErrTokenMissing
	}
	return xToken, nil
}

func isTestEnv() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("INFRA_ENV")), config.TEST)
}

// ParseTokenWithFallback tries nexo token first, then falls back to external token if enabled.
func ParseTokenWithFallback(tokenString string, cfg *config.Config) (*jwt.Claims, error) {
	if cfg == nil {
		return nil, errcode.ErrTokenInvalid
	}

	// Try nexo native token first
	claims, err := jwt.ParseToken(tokenString, cfg.JWT.Secret)
	if err == nil {
		return claims, nil
	}

	// Fall back to external token if enabled
	if cfg.ExternalJWT.Enabled {
		return jwt.ParseExternalToken(
			tokenString,
			cfg.ExternalJWT.Secret,
			cfg.ExternalJWT.DefaultRole,
			cfg.ExternalJWT.DefaultPlatformId,
		)
	}

	return nil, err
}

// GetUserId gets user Id from context
func GetUserId(c *app.RequestContext) string {
	if v, ok := c.Get(UserIdKey); ok {
		return v.(string)
	}
	return ""
}

// GetPlatformId gets platform Id from context
func GetPlatformId(c *app.RequestContext) int {
	if v, ok := c.Get(PlatformIdKey); ok {
		return v.(int)
	}
	return 0
}
