package service

import (
	"context"
	"errors"
	"strings"
	"github.com/google/uuid"
	repo "uas/app/repository/postgres" // Asumsi path repository
	models "uas/app/model/postgres"    // Asumsi path model
)

// StudentService interface
type StudentService interface {
	ListStudents(ctx context.Context) ([]models.Student, error)
	GetStudentDetail(ctx context.Context, studentID string) (*models.Student, error)
	UpdateAdvisor(ctx context.Context, studentID string, newAdvisorID string, callerRole string) error // Hanya Admin
}

type StudentServiceImpl struct {
	Repo repo.StudentRepository
}

func NewStudentService(repo repo.StudentRepository) StudentService {
	return &StudentServiceImpl{Repo: repo}
}

// Implementasi ListStudents
func (s *StudentServiceImpl) ListStudents(ctx context.Context) ([]models.Student, error) {
	return s.Repo.GetAll(ctx)
}

// Implementasi GetStudentDetail
func (s *StudentServiceImpl) GetStudentDetail(ctx context.Context, studentIDStr string) (*models.Student, error) {
	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		return nil, errors.New("invalid student ID format (expected UUID)")
	}
	return s.Repo.GetByID(ctx, studentID)
}

// Implementasi UpdateAdvisor (Hanya Admin yang diizinkan mengganti dosen wali)
func (s *StudentServiceImpl) UpdateAdvisor(ctx context.Context, studentIDStr string, newAdvisorIDStr string, callerRole string) error {

    if callerRole != "admin" {
        return errors.New("forbidden: only admin can update advisor")
    }

    // 👇 WAJIB TAMBAH INI
    studentIDStr = strings.TrimSpace(studentIDStr)
    newAdvisorIDStr = strings.TrimSpace(newAdvisorIDStr)

    studentID, err := uuid.Parse(studentIDStr)
    if err != nil {
        return errors.New("invalid student ID format")
    }

    newAdvisorID, err := uuid.Parse(newAdvisorIDStr)
    if err != nil {
        return errors.New("invalid advisor ID format")
    }

    return s.Repo.UpdateAdvisor(ctx, studentID, newAdvisorID)
}
