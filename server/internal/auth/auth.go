// Package auth implements user registration, login, JWT signing/verification
// and the HTTP middleware that protects authenticated routes.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors returned by the service.
var (
	ErrUsernameTaken       = errors.New("username taken")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidUsername     = errors.New("username must be 3-32 characters")
	ErrInvalidPassword     = errors.New("password must be 6-128 characters")
)

// Service handles auth persistence and JWT operations.
type Service struct {
	Pool   *pgxpool.Pool
	Secret []byte
	TTL    time.Duration
}

// Claims is the JWT payload.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	jwt.RegisteredClaims
}

type ctxKey string

const (
	CtxUserID   ctxKey = "uid"
	CtxUsername ctxKey = "usr"
)

// New constructs an auth Service.
func New(pool *pgxpool.Pool, secret string, ttl time.Duration) *Service {
	return &Service{Pool: pool, Secret: []byte(secret), TTL: ttl}
}

// ValidateCredentials checks username/password length constraints.
func ValidateCredentials(username, password string) error {
	if len(username) < 3 || len(username) > 32 {
		return ErrInvalidUsername
	}
	if len(password) < 6 || len(password) > 128 {
		return ErrInvalidPassword
	}
	return nil
}

// Register creates a new user (and its stats row) and returns a fresh JWT.
func (s *Service) Register(ctx context.Context, username, password string) (token string, userID int64, exp int64, err error) {
	if err := ValidateCredentials(username, password); err != nil {
		return "", 0, 0, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", 0, 0, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", 0, 0, err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx,
		"INSERT INTO users(username, password_hash) VALUES($1,$2) RETURNING id",
		username, string(hash)).Scan(&userID); err != nil {
		if isUniqueViolation(err) {
			return "", 0, 0, ErrUsernameTaken
		}
		return "", 0, 0, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO user_stats(user_id) VALUES($1)", userID); err != nil {
		return "", 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", 0, 0, err
	}
	return s.sign(userID, username)
}

// Login verifies credentials and returns a fresh JWT.
func (s *Service) Login(ctx context.Context, username, password string) (token string, userID int64, exp int64, err error) {
	var hash string
	err = s.Pool.QueryRow(ctx,
		"SELECT id, password_hash FROM users WHERE username=$1", username).Scan(&userID, &hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, 0, ErrInvalidCredentials
		}
		return "", 0, 0, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", 0, 0, ErrInvalidCredentials
	}
	return s.sign(userID, username)
}

func (s *Service) sign(userID int64, username string) (string, int64, error) {
	now := time.Now()
	exp := now.Add(s.TTL)
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   username,
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := t.SignedString(s.Secret)
	if err != nil {
		return "", 0, err
	}
	return tokenStr, exp.Unix(), nil
}

// Verify parses and validates a JWT, returning its claims.
func (s *Service) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.Secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// Middleware protects HTTP routes with a Bearer JWT check.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}
		tokenStr := strings.TrimPrefix(authz, "Bearer ")
		claims, err := s.Verify(tokenStr)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
		ctx = context.WithValue(ctx, CtxUsername, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromCtx extracts the authenticated user id from a request context.
func UserIDFromCtx(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(CtxUserID).(int64)
	return v, ok
}

// UsernameFromCtx extracts the authenticated username from a request context.
func UsernameFromCtx(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(CtxUsername).(string)
	return v, ok
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
