package model

import (
	"time"

	"github.com/google/uuid"
)

type AchievementStatus string

const (
	StatusDraft     AchievementStatus = "draft"
	StatusSubmitted AchievementStatus = "submitted"
	StatusVerified  AchievementStatus = "verified"
	StatusRejected  AchievementStatus = "rejected"
)

type AchievementReference struct {
	ID                 uuid.UUID         `json:"id" db:"id"`
	StudentID          uuid.UUID         `json:"student_id" db:"student_id"`
	MongoAchievementID string            `json:"mongo_achievement_id" db:"mongo_achievement_id"`
	Status             AchievementStatus `json:"status" db:"status"`
	SubmittedAt        *time.Time        `json:"submitted_at,omitempty" db:"submitted_at"`
	VerifiedAt         *time.Time        `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy         *uuid.UUID        `json:"verified_by,omitempty" db:"verified_by"`
	RejectionNote      *string           `json:"rejection_note,omitempty" db:"rejection_note"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at" db:"updated_at"`
}
