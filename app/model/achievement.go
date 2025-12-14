package model

import (
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Definisi ENUM untuk Status Prestasi
// Ini merepresentasikan ENUM('draft', 'submitted', 'verified', 'rejected') di Database
type AchievementStatus string

const (
	StatusDraft     AchievementStatus = "draft"
	StatusSubmitted AchievementStatus = "submitted"
	StatusVerified  AchievementStatus = "verified"
	StatusRejected  AchievementStatus = "rejected"
)

// --- PostgreSQL Model (Reference) ---
type AchievementReference struct {
	ID                 uuid.UUID         `json:"id"`
	StudentID          uuid.UUID         `json:"student_id"`
	MongoAchievementID string            `json:"mongo_achievement_id"`
	Status             AchievementStatus `json:"status"` // Menggunakan tipe custom ENUM
	SubmittedAt        *time.Time        `json:"submitted_at"`
	VerifiedAt         *time.Time        `json:"verified_at"`
	VerifiedBy         *uuid.UUID        `json:"verified_by"`
	RejectionNote      string            `json:"rejection_note"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	DeletedAt *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
}

// --- MongoDB Model (Dynamic Data) ---
type AchievementMongo struct {
	ID              primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	StudentID       string                 `bson:"studentId" json:"studentId"`
	AchievementType string                 `bson:"achievementType" json:"achievementType"`
	Title           string                 `bson:"title" json:"title"`
	Description     string                 `bson:"description" json:"description"`
	Details         map[string]interface{} `bson:"details" json:"details"`
	Attachments     []Attachment           `bson:"attachments" json:"attachments"`
	Tags            []string               `bson:"tags" json:"tags"`
	Points          int                    `bson:"points" json:"points"`
	CreatedAt       time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time              `bson:"updatedAt" json:"updatedAt"`
	DeletedAt 		*time.Time 				`bson:"deleted_at,omitempty"`
}

type Attachment struct {
	FileName   string    `bson:"fileName" json:"fileName"`
	FileURL    string    `bson:"fileUrl" json:"fileUrl"`
	FileType   string    `bson:"fileType" json:"fileType"`
	UploadedAt time.Time `bson:"uploadedAt" json:"uploadedAt"`
}