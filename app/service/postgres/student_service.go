package services

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
	GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type StudentServiceImpl struct {
	Repo repo.StudentRepository
}

func NewStudentService(repo repo.StudentRepository) StudentService {
	return &StudentServiceImpl{Repo: repo}
}

func (s *StudentServiceImpl) ListStudents(
	ctx context.Context,
	requestingUserID uuid.UUID,
	requestingUserRole string,
) ([]models.Student, error) {

	role := strings.ToLower(strings.TrimSpace(requestingUserRole))

	if role == "admin" || role == "administrator" {
		return s.Repo.GetAll(ctx)
	}

	if role == "student" || role == "mahasiswa" {
		student, err := s.Repo.FindStudentByUserID(ctx, requestingUserID)
		if err != nil {
			return nil, errors.New("student data not found for this user ID")
		}
		return []models.Student{*student}, nil
	}

	return nil, errors.New("unauthorized: only admin or the student themself can view this list")
}

func (s *StudentServiceImpl) GetStudentDetail(ctx context.Context, studentIDStr string) (*models.Student, error) {
	studentIDStr = strings.TrimSpace(studentIDStr)

	studentID, err := uuid.Parse(studentIDStr)
	if err != nil {
		return nil, errors.New("invalid student ID format (expected UUID)")
	}

	return s.Repo.GetByID(ctx, studentID)
}

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
func (s *StudentServiceImpl) GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
    // ambil profileID dari DB
    profileID, err := s.Repo.GetStudentIDByUserID(ctx, userID)
    if err != nil {
        return uuid.Nil, err
    }
    return profileID, nil
}