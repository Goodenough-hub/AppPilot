package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const TokenTTL = 24 * time.Hour

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
