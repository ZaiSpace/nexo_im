package jwt

import (
	"github.com/golang-jwt/jwt/v5"

	"github.com/mbeoliero/nexo/pkg/errcode"
)

// ExternalClaims 另一个系统的 claims 结构
type ExternalClaims struct {
	// 根据另一个系统的实际结构调整
	UserId string `json:"user_id"` // 或 uid, sub 等
	jwt.RegisteredClaims
}

// ParseMultiIssuer 支持多个系统的 token
func ParseMultiIssuer(tokenString, nexoSecret, externalSecret string) (*Claims, error) {
	// 先尝试当前系统的格式
	if claims, err := ParseToken(tokenString, nexoSecret); err == nil {
		return claims, nil
	}

	// 尝试外部系统的格式
	token, err := jwt.ParseWithClaims(tokenString, &ExternalClaims{}, func(token *jwt.Token) (any, error) {
		// 根据 issuer 判断使用哪个密钥
		if claims, ok := token.Claims.(*ExternalClaims); ok {
			if claims.Issuer == "external-system" {
				return []byte(externalSecret), nil
			}
		}
		return []byte(nexoSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if extClaims, ok := token.Claims.(*ExternalClaims); ok && token.Valid {
		// 转换为当前系统的 Claims 格式
		return &Claims{
			UserId:           extClaims.UserId,
			PlatformId:       1, // 默认平台或从其他字段映射
			RegisteredClaims: extClaims.RegisteredClaims,
		}, nil
	}

	return nil, errcode.ErrTokenInvalid
}
