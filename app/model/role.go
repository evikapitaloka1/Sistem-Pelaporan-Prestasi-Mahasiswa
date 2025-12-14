package model

import (
	"time"

	"github.com/google/uuid"
)

// SRS 3.1.2 Tabel roles
type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
}