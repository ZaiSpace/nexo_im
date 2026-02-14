package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/ZaiSpace/nexo_im/internal/config"
	"github.com/ZaiSpace/nexo_im/pkg/errcode"
	"github.com/ZaiSpace/nexo_im/pkg/response"
)

const (
	InternalServiceNameHeader = "X-Service-Name"
	InternalTimestampHeader   = "X-Timestamp"
	InternalSignatureHeader   = "X-Signature"
	InternalUserIdHeader      = "X-User-Id"
	InternalPlatformIdHeader  = "X-Platform-Id"
	InternalServiceNameKey    = "internal_service_name"
)

// InternalAuth validates service-to-service requests using:
// X-Service-Name + X-Timestamp + X-Signature.
func InternalAuth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		serviceName, authErr := validateInternalRequest(c)
		if authErr != nil {
			response.ErrorWithCode(ctx, c, authErr)
			c.Abort()
			return
		}
		c.Set(InternalServiceNameKey, serviceName)
		c.Next(ctx)
	}
}

// InternalAuthAsUser validates internal auth and injects user context.
func InternalAuthAsUser() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		serviceName, authErr := validateInternalRequest(c)
		if authErr != nil {
			response.ErrorWithCode(ctx, c, authErr)
			c.Abort()
			return
		}

		userId := strings.TrimSpace(string(c.GetHeader(InternalUserIdHeader)))
		if userId == "" {
			response.ErrorWithCode(ctx, c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}

		platformId := 5 // Web default
		platformIdStr := strings.TrimSpace(string(c.GetHeader(InternalPlatformIdHeader)))
		if platformIdStr != "" {
			pid, err := strconv.Atoi(platformIdStr)
			if err != nil || pid <= 0 {
				response.ErrorWithCode(ctx, c, errcode.ErrInvalidParam)
				c.Abort()
				return
			}
			platformId = pid
		}

		c.Set(InternalServiceNameKey, serviceName)
		c.Set(UserIdKey, userId)
		c.Set(PlatformIdKey, platformId)
		c.Next(ctx)
	}
}

func validateInternalRequest(c *app.RequestContext) (string, *errcode.Error) {
	cfg := config.GlobalConfig
	if cfg == nil || !cfg.InternalAuth.Enabled {
		return "", errcode.ErrForbidden
	}
	if strings.TrimSpace(cfg.InternalAuth.Secret) == "" {
		return "", errcode.ErrForbidden
	}

	serviceName := strings.TrimSpace(string(c.GetHeader(InternalServiceNameHeader)))
	tsStr := strings.TrimSpace(string(c.GetHeader(InternalTimestampHeader)))
	signature := strings.TrimSpace(string(c.GetHeader(InternalSignatureHeader)))
	if serviceName == "" || tsStr == "" || signature == "" {
		return "", errcode.ErrUnauthorized
	}

	if !isServiceAllowed(serviceName, cfg.InternalAuth.AllowedServices) {
		return "", errcode.ErrForbidden
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", errcode.ErrUnauthorized
	}

	now := time.Now().Unix()
	if absInt64(now-ts) > cfg.InternalAuth.MaxSkewSeconds {
		return "", errcode.ErrUnauthorized
	}

	body := c.Request.Body()
	expected := signInternalRequest(
		cfg.InternalAuth.Secret,
		serviceName,
		tsStr,
		string(c.Method()),
		string(c.Path()),
		body,
	)
	if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
		return "", errcode.ErrUnauthorized
	}
	return serviceName, nil
}

func signInternalRequest(secret, serviceName, timestamp, method, path string, body []byte) string {
	bodyHashBytes := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(bodyHashBytes[:])
	payload := strings.Join([]string{
		serviceName,
		timestamp,
		strings.ToUpper(method),
		path,
		bodyHash,
	}, "\n")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func isServiceAllowed(serviceName string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, s := range allowed {
		if strings.EqualFold(strings.TrimSpace(s), serviceName) {
			return true
		}
	}
	return false
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// GetInternalServiceName returns the calling service name from context.
func GetInternalServiceName(c *app.RequestContext) string {
	if v, ok := c.Get(InternalServiceNameKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
