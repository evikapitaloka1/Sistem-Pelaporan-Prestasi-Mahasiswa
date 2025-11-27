package model

import (
	"time"
	"github.com/google/uuid"
)

type Lecturer struct {
	
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"userId"`
	LecturerID   string    `json:"lecturerId"`
	Department   string    `json:"department"`
	CreatedAt    time.Time `json:"createdAt"`
}