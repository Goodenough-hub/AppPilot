package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenTTL = 24 * time.Hour

const RefreshGraceTTL = 7 * 24 * time.Hour

type Claims struct {
	UserID   int64    `json:"uid"`
	Role     string   `json:"role"`
	AppScope []string `json:"scope"`
	Username string   `json:"usr"`
	jwt.RegisteredClaims
}

func GenerateToken(u *User, secret string) (string, int64, error) {
	exp := time.Now().Add(TokenTTL)
	claims := Claims{
		UserID:   u.ID,
		Role:     u.Role,
		AppScope: u.AppScope,
		Username: u.Username,
		RegisteredClaims: jwt.RegisteredClaims{
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

func ParseToken(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ParseTokenForRefresh 解析 token 用于刷新场景：允许过期，但签发时间必须在
// RefreshGraceTTL 内（防止无限期 refresh）。签名错误或非 HMAC 仍返回 error。
func ParseTokenForRefresh(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
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
