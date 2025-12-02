package model

import (
	"time"
	"github.com/google/uuid"
)

type Student struct {
	
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"userId"`
	StudentID      string    `json:"studentId"`
	ProgramStudy   string    `json:"programStudy"`
	AcademicYear   string    `json:"academicYear"`
	AdvisorID      uuid.UUID `json:"advisorId"`
	CreatedAt      time.Time `json:"createdAt"`
}
type UpdateAdvisorRequest struct {
    NewAdvisorID string `json:"new_advisor_id"`
}
