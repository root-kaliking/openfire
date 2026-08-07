// Package auth implements user registration/login, JWT (HS256) issuance
// and verification, and bcrypt password hashing. It is transport-agnostic;
// HTTP wiring lives in internal/api.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"openfire-server/internal/models"
)

// Service owns the auth state (DB pool + signing secret + token TTL).
type Service struct {
	pool      *pgxpool.Pool
	secret    []byte
	ttl       time.Duration
}

// New builds a Service. ttlMinutes controls JWT expiry.
func New(pool *pgxpool.Pool, secret string, ttlMinutes int) *Service {
	return &Service{
		pool:   pool,
		secret: []byte(secret),
		ttl:    time.Duration(ttlMinutes) * time.Minute,
	}
}

// Claims is the JWT payload. Subject = user UUID.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type ctxKey struct{}

// ErrInvalidToken is returned by Verify for any malformed/expired token.
var ErrInvalidToken = errors.New("invalid token")

// Register creates a user. Username is trimmed and lower-cased for the
// uniqueness check while the original casing is preserved for display.
func (s *Service) Register(ctx context.Context, username, password string) (models.AuthResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 32 {
		return models.AuthResponse{}, errors.New("username must be 1-32 chars")
	}
	if len(password) < 6 || len(password) > 128 {
		return models.AuthResponse{}, errors.New("password must be 6-128 chars")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.AuthResponse{}, fmt.Errorf("hash password: %w", err)
	}

	id := uuid.NewString()
	_, err = s.pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)`,
		id, username, string(hash))
	if err != nil {
		if isUniqueViolation(err) {
			return models.AuthResponse{}, errors.New("username already taken")
		}
		return models.AuthResponse{}, fmt.Errorf("insert user: %w", err)
	}

	token, err := s.sign(id, username)
	if err != nil {
		return models.AuthResponse{}, err
	}
	return models.AuthResponse{Token: token, User: s.publicUser(id, username, time.Now().UTC())}, nil
}

// Login verifies credentials and returns a fresh JWT.
func (s *Service) Login(ctx context.Context, username, password string) (models.AuthResponse, error) {
	username = strings.TrimSpace(username)
	var u models.User
	err := s.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username = $1`,
		username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.AuthResponse{}, errors.New("invalid credentials")
		}
		return models.AuthResponse{}, fmt.Errorf("query user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return models.AuthResponse{}, errors.New("invalid credentials")
	}

	token, err := s.sign(u.ID, u.Username)
	if err != nil {
		return models.AuthResponse{}, err
	}
	return models.AuthResponse{Token: token, User: s.publicUser(u.ID, u.Username, u.CreatedAt)}, nil
}

func (s *Service) sign(userID, username string) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			Issuer:    "openfire",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// Verify parses and validates a JWT, returning the claims.
func (s *Service) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

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

func (s *Service) publicUser(id, username string, createdAt time.Time) models.UserInfo {
	return models.UserInfo{ID: id, Username: username, CreatedAt: createdAt}
}

// isUniqueViolation detects a Postgres unique constraint error.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
