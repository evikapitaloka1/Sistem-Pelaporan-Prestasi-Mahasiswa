package model

import (
	"time"

	"github.com/google/uuid"
)

// Specific Profile for Lecturer
type Lecturer struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	LecturerID string    `json:"lecturer_id"` // NIP/NIDN
	Department string    `json:"department"`
	CreatedAt  time.Time `json:"created_at"`
}