package model

import (
	"database/sql" 
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- MONGODB Achievement Structs (Data Dinamis) ---

// Attachment merepresentasikan file pendukung.
type Attachment struct {
    FileName string `json:"fileName" bson:"fileName"`
    FileUrl string `json:"fileUrl" bson:"fileUrl"`
    FileType string `json:"fileType" bson:"fileType"`
    UploadedAt time.Time `json:"uploadedAt" bson:"uploadedAt"`
}

// Period digunakan untuk rentang waktu dalam Organization Details.
type Period struct {
    Start time.Time `json:"start" bson:"start"`
    End time.Time `json:"end" bson:"end"`
}

// DynamicDetails menampung semua field dinamis.
type DynamicDetails struct {
    // Competition
    CompetitionName string `json:"competitionName,omitempty" bson:"competitionName,omitempty"`
    CompetitionLevel string `json:"competitionLevel,omitempty" bson:"competitionLevel,omitempty"`
    Rank int `json:"rank,omitempty" bson:"rank,omitempty"`
    MedalType string `json:"medalType,omitempty" bson:"medalType,omitempty"`
    // Publication
    PublicationType string `json:"publicationType,omitempty" bson:"publicationType,omitempty"`
    PublicationTitle string `json:"publicationTitle,omitempty" bson:"publicationTitle,omitempty"`
    Authors []string `json:"authors,omitempty" bson:"authors,omitempty"`
    Publisher string `json:"publisher,omitempty" bson:"publisher,omitempty"`
    ISSN string `json:"issn,omitempty" bson:"issn,omitempty"`
    // Organization
    OrganizationName string `json:"organizationName,omitempty" bson:"organizationName,omitempty"`
    Position string `json:"position,omitempty" bson:"position,omitempty"`
    Period Period `json:"period,omitempty" bson:"period,omitempty"`
    // Certification
    CertificationName string `json:"certificationName,omitempty" bson:"certificationName,omitempty"`
    IssuedBy string `json:"issuedBy,omitempty" bson:"issuedBy,omitempty"`
    CertificationNumber string `json:"certificationNumber,omitempty" bson:"certificationNumber,omitempty"`
    ValidUntil time.Time `json:"validUntil,omitempty" bson:"validUntil,omitempty"`
    // Common Fields
    EventDate time.Time `json:"eventDate,omitempty" bson:"eventDate,omitempty"`
    Location string `json:"location,omitempty" bson:"location,omitempty"`
    Organizer string `json:"organizer,omitempty" bson:"organizer,omitempty"`
    Score float64 `json:"score,omitempty" bson:"score,omitempty"`
    CustomFields primitive.M `json:"customFields,omitempty" bson:"customFields,omitempty"`
}

// Achievement adalah model utama untuk dokumen prestasi di MongoDB.
type Achievement struct {
    ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
    StudentUUID string `json:"studentId" bson:"studentId"` // Ref ke students.id
    AchievementType string `json:"achievementType" bson:"achievementType"`
    Title string `json:"title" bson:"title"`
    Description string `json:"description" bson:"description"`
    Details DynamicDetails `json:"details" bson:"details"`
    Attachments []Attachment `json:"attachments" bson:"attachments"`
    Tags []string `json:"tags" bson:"tags"`
    Points float64 `json:"points" bson:"points"`
    CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
    DeletedAt *time.Time `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// AchievementRequest digunakan untuk input dari Mahasiswa.
type AchievementRequest struct {
    AchievementType string `json:"achievementType" validate:"required"`
    Title string `json:"title" validate:"required,max=255"`
    Description string `json:"description" validate:"required"`
    Details DynamicDetails `json:"details"`
    Tags []string `json:"tags"`
    Points float64 `json:"points"`
    TargetStudentID string `json:"targetStudentId,omitempty"`
}

// --- POSTGRESQL Structs (Data Relasional/Workflow) ---

// AchievementStatus merepresentasikan status verifikasi di PostgreSQL.
type AchievementStatus string

const (
    StatusDraft AchievementStatus = "draft"
    StatusSubmitted AchievementStatus = "submitted"
    StatusVerified AchievementStatus = "verified"
    StatusRejected AchievementStatus = "rejected"
)

// AchievementReference merepresentasikan tabel achievement_references di PostgreSQL.
type AchievementReference struct {
    ID uuid.UUID `json:"id" db:"id"`
    StudentID uuid.UUID `json:"student_id" db:"student_id"`
    MongoAchievementID string `json:"mongo_achievement_id" db:"mongo_achievement_id"` // Jembatan ke MongoDB
    Status AchievementStatus `json:"status" db:"status"`
    
    // --- FIELD WORKFLOW (NULLABLE) YANG DIPERBAIKI ---
    SubmittedAt sql.NullTime `json:"submitted_at,omitempty" db:"submitted_at"`
    VerifiedAt sql.NullTime `json:"verified_at,omitempty" db:"verified_at"`
    VerifiedBy *uuid.UUID `json:"verified_by,omitempty" db:"verified_by"` // <-- FIX: Gunakan pointer *uuid.UUID untuk kolom UUID nullable
    RejectionNote sql.NullString `json:"rejection_note,omitempty" db:"rejection_note"`
    // --- AKHIR FIELD WORKFLOW ---

    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AchievementDetail adalah gabungan data dari MongoDB dan PostgreSQL untuk ditampilkan ke client.
type AchievementDetail struct {
    Achievement Achievement `json:"achievement"`
    ReferenceID string `json:"referenceId"`
    Status AchievementStatus `json:"status"`
    SubmittedAt *time.Time `json:"submittedAt,omitempty"`
    VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
    VerifiedBy string `json:"verifiedBy,omitempty"`
    RejectionNote string `json:"rejectionNote,omitempty"`
}