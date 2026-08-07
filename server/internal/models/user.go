// Package models defines the database row structs shared across the
// central server. API request/response payloads live next to the
// handlers that own them.
package models

import "time"

// User is the users table row. PasswordHash is never serialized.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
