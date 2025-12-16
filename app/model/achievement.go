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
// --- PostgreSQL Model (Reference) ---
type AchievementReference struct {
	ID                 uuid.UUID         `db:"id" json:"id"`
	StudentID          uuid.UUID         `db:"student_id" json:"student_id"`
	MongoAchievementID string            `db:"mongo_achievement_id" json:"mongo_achievement_id"`
	Status             AchievementStatus `db:"status" json:"status"`

	// Pointer fields (nullable)
	SubmittedAt   *time.Time `db:"submitted_at" json:"submitted_at"`
	VerifiedAt    *time.Time `db:"verified_at" json:"verified_at"`
	VerifiedBy    *uuid.UUID `db:"verified_by" json:"verified_by"`
	RejectionNote *string    `db:"rejection_note" json:"rejection_note"`
	DeletedAt     *time.Time `db:"deleted_at" json:"deleted_at"`

	// Non-nullable
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Nullable
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
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
	DeletedAt 		*time.Time `db:"deleted_at" json:"deleted_at"`
}

type Attachment struct {
	FileName   string    `bson:"fileName" json:"fileName"`
	FileURL    string    `bson:"fileUrl" json:"fileUrl"`
	FileType   string    `bson:"fileType" json:"fileType"`
	UploadedAt time.Time `bson:"uploadedAt" json:"uploadedAt"`
}
type RejectRequest struct {
	RejectionNote string `json:"rejection_note"` // Catatan penolakan
}