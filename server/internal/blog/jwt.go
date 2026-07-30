package blog

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// blog JWT 与 finflow/admin 的 auth.JWTSecret 体系完全隔离：
// - 独立 secret（cfg.BlogJWTSecret）
// - 固定 iss=apppilot、aud=fluxblog
// - 携带 token_version，停用/删除账号时递增使旧令牌失效
// - access 2h；可在签发后 7 天内 refresh
const (
	blogIssuer      = "apppilot"
	blogAudience    = "fluxblog"
	AccessTokenTTL  = 2 * time.Hour
	RefreshGraceTTL = 7 * 24 * time.Hour
)

type Claims struct {
	UserID       int64  `json:"uid"`
	Username     string `json:"usr"`
	TokenVersion int64  `json:"tv"`
	jwt.RegisteredClaims
}

// GenerateToken 为 blog 账号签发 access token。
func GenerateToken(u *BlogUser, secret string) (string, int64, error) {
	if len(secret) < 32 {
		return "", 0, errors.New("blog jwt secret too short")
	}
	exp := time.Now().Add(AccessTokenTTL)
	claims := Claims{
		UserID:       u.ID,
		Username:     u.Username,
		TokenVersion: u.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    blogIssuer,
			Audience:  jwt.ClaimStrings{blogAudience},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return s, exp.Unix(), nil
}

type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	UserID    string `json:"userId"`
	Username  string `json:"username"`
}

// ParseToken 校验签名、iss、aud；过期视为无效（用于受保护接口）。
func ParseToken(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	},
		jwt.WithIssuer(blogIssuer),
		jwt.WithAudience(blogAudience),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ParseTokenForRefresh 允许 token 过期，但签发时间必须在 RefreshGraceTTL 内。
// iss/aud 仍需匹配；签名错误或非 HMAC 返回 error。
func ParseTokenForRefresh(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	},
		jwt.WithIssuer(blogIssuer),
		jwt.WithAudience(blogAudience),
	)
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return nil, err
	}
	if claims.UserID == 0 {
		return nil, errors.New("invalid claims")
	}
	if claims.IssuedAt == nil {
		return nil, errors.New("missing iat")
	}
	if time.Since(claims.IssuedAt.Time) > RefreshGraceTTL {
		return nil, errors.New("refresh window expired")
	}
	return claims, nil
}
