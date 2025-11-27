package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Period struct {
	Start time.Time `bson:"start,omitempty" json:"start,omitempty"`
	End   time.Time `bson:"end,omitempty" json:"end,omitempty"`
}

type Attachment struct {
	FileName   string    `bson:"fileName" json:"fileName"`
	FileUrl    string    `bson:"fileUrl" json:"fileUrl"`
	FileType   string    `bson:"fileType,omitempty" json:"fileType,omitempty"`
	UploadedAt time.Time `bson:"uploadedAt,omitempty" json:"uploadedAt,omitempty"`
}

type AchievementDetails struct {
	CompetitionName  string `bson:"competitionName,omitempty" json:"competitionName,omitempty"`
	CompetitionLevel string `bson:"competitionLevel,omitempty" json:"competitionLevel,omitempty"`
	Rank             int    `bson:"rank,omitempty" json:"rank,omitempty"`
	MedalType        string `bson:"medalType,omitempty" json:"medalType,omitempty"`
	PublicationType  string   `bson:"publicationType,omitempty" json:"publicationType,omitempty"`
	PublicationTitle string   `bson:"publicationTitle,omitempty" json:"publicationTitle,omitempty"`
	Authors          []string `bson:"authors,omitempty" json:"authors,omitempty"`
	Publisher        string   `bson:"publisher,omitempty" json:"publisher,omitempty"`
	ISSN             string   `bson:"issn,omitempty" json:"issn,omitempty"`
	OrganizationName string  `bson:"organizationName,omitempty" json:"organizationName,omitempty"`
	Position         string  `bson:"position,omitempty" json:"position,omitempty"`
	Period           *Period `bson:"period,omitempty" json:"period,omitempty"`
	CertificationName   string    `bson:"certificationName,omitempty" json:"certificationName,omitempty"`
	IssuedBy            string    `bson:"issuedBy,omitempty" json:"issuedBy,omitempty"`
	CertificationNumber string    `bson:"certificationNumber,omitempty" json:"certificationNumber,omitempty"`
	ValidUntil          time.Time `bson:"validUntil,omitempty" json:"validUntil,omitempty"`
	EventDate time.Time `bson:"eventDate,omitempty" json:"eventDate,omitempty"`
	Location  string    `bson:"location,omitempty" json:"location,omitempty"`
	Organizer string    `bson:"organizer,omitempty" json:"organizer,omitempty"`
	Score     float64   `bson:"score,omitempty" json:"score,omitempty"`

	CustomFields map[string]interface{} `bson:"customFields,omitempty" json:"customFields,omitempty"`
}

type Achievement struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID string `bson:"studentId" json:"studentId"`
	AchievementType string `bson:"achievementType" json:"achievementType"`
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Details AchievementDetails `bson:"details,omitempty" json:"details,omitempty"`
	Attachments []Attachment `bson:"attachments,omitempty" json:"attachments,omitempty"`
	Tags   []string `bson:"tags,omitempty" json:"tags,omitempty"`
	Points int      `bson:"points,omitempty" json:"points,omitempty"`
	CreatedAt time.Time `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}
