package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	repo "uas/app/repository/postgres"
	models "uas/app/model/postgres"
)

type StudentService interface {
	ListStudents(ctx context.Context, requestingUserID uuid.UUID, requestingUserRole string) ([]models.Student, error)
	GetStudentDetail(ctx context.Context, studentID string) (*models.Student, error)
	UpdateAdvisor(ctx context.Context, studentID string, newAdvisorID string, callerRole string) error
}

type StudentServiceImpl struct {
	Repo repo.StudentRepository
}

func NewStudentService(repo repo.StudentRepository) StudentService {
	return &StudentServiceImpl{Repo: repo}
}

//
// ========================================================================
// ListStudents — Mengandung logika RBAC: Admin full access, mahasiswa self-access
// ========================================================================
func (s *StudentServiceImpl) ListStudents(
	ctx context.Context,
	requestingUserID uuid.UUID,
	requestingUserRole string,
) ([]models.Student, error) {

	role := strings.ToLower(strings.TrimSpace(requestingUserRole))

	// --- 1. Admin memiliki akses penuh ---
	if role == "admin" || role == "administrator" {
		return s.Repo.GetAll(ctx)
	}

	// --- 2. Mahasiswa hanya bisa melihat data dirinya sendiri ---
	if role == "student" || role == "mahasiswa" {
		student, err := s.Repo.FindStudentByUserID(ctx, requestingUserID)
		if err != nil {
			return nil, errors.New("student data not found for this user ID")
		}
		return []models.Student{*student}, nil
	}

	// --- 3. Role lain (dosen, operator, dll) tidak boleh akses ---
	return nil, errors.New("unauthorized: only admin or the student themself can view this list")
}

//
// ========================================================================
// GetStudentDetail — Mengambil detail mahasiswa berdasarkan ID UUID
// ========================================================================
func (s *StudentServiceImpl) GetStudentDetail(ctx context.Context, studentIDStr string) (*models.Student, error) {
	studentIDStr = strings.TrimSpace(studentIDStr)

	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		return nil, errors.New("invalid student ID format (expected UUID)")
	}

	return s.Repo.GetByID(ctx, studentID)
}

//
// ========================================================================
// UpdateAdvisor — Hanya Admin yang boleh update advisor mahasiswa
// ========================================================================
func (s *StudentServiceImpl) UpdateAdvisor(ctx context.Context, studentIDStr string, newAdvisorIDStr string, callerRole string) error {

	if strings.ToLower(callerRole) != "admin" {
		return errors.New("forbidden: only admin can update advisor")
	}

	studentIDStr = strings.TrimSpace(studentIDStr)
	newAdvisorIDStr = strings.TrimSpace(newAdvisorIDStr)

	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		return errors.New("invalid student ID format (expected UUID)")
	}

	newAdvisorID, err := uuid.Parse(newAdvisorIDStr)
	if err != nil {
		return errors.New("invalid advisor ID format (expected UUID)")
	}

	return s.Repo.UpdateAdvisor(ctx, studentID, newAdvisorID)
}
