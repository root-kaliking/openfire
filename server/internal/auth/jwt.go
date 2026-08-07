// Package auth implements JWT (HS256) issuance/verification, bcrypt
// password hashing, and context helpers for propagating the authenticated
// user through the request lifecycle.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload. Subject holds the user UUID.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// ErrInvalidToken is returned by Verify for any malformed/expired token.
var ErrInvalidToken = errors.New("invalid token")

// Sign issues a signed HS256 JWT for the user that expires after ttl.
func Sign(secret, userID, username string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Issuer:    "openfire",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", errors.New("sign jwt")
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning its claims.
func Verify(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

type ctxKey struct{}

// ContextWithUser stores the verified user id/username in ctx.
func ContextWithUser(ctx context.Context, userID, username string) context.Context {
	return context.WithValue(ctx, ctxKey{}, [2]string{userID, username})
}

// UserFromContext returns (userID, username, ok).
func UserFromContext(ctx context.Context) (string, string, bool) {
	v, ok := ctx.Value(ctxKey{}).([2]string)
	if !ok {
		return "", "", false
	}
	return v[0], v[1], true
}
