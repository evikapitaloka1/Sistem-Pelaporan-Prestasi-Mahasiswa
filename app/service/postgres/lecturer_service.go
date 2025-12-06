package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	model "uas/app/model/postgres"
	lecturerRepo "uas/app/repository/postgres"
)

type LecturerService interface {
	GetAllLecturers(ctx context.Context, requestingUserID uuid.UUID, requestingUserRole string) ([]model.Lecturer, error)
	GetAdvisees(ctx context.Context, targetLecturerID uuid.UUID, requestingUserID uuid.UUID, requestingUserRole string) ([]model.Advisee, error)
}

type lecturerSvc struct {
	repo lecturerRepo.LecturerRepository
}

func NewLecturerService(repo lecturerRepo.LecturerRepository) LecturerService {
	return &lecturerSvc{repo: repo}
}

func (s *lecturerSvc) GetAllLecturers(
	ctx context.Context,
	requestingUserID uuid.UUID,
	requestingUserRole string,
) ([]model.Lecturer, error) {

	normalizedRole := strings.ToLower(requestingUserRole)

	if normalizedRole == "admin" {
		return s.repo.ListAllLecturers(ctx)
	}

	isLecturer := normalizedRole == "dosen wali" || normalizedRole == "lecturer" || normalizedRole == "pengajar"

	if isLecturer {
		lecturer, err := s.repo.FindLecturerByUserID(ctx, requestingUserID)
		if err != nil {
			return nil, errors.New("lecturer data not found for this user")
		}

		return []model.Lecturer{*lecturer}, nil
	}

	return nil, errors.New("unauthorized: only lecturers and administrators can view this list")
}

func (s *lecturerSvc) GetAdvisees(
	ctx context.Context,
	targetLecturerID uuid.UUID,
	requestingUserID uuid.UUID,
	requestingUserRole string,
) ([]model.Advisee, error) {

	normalizedRole := strings.ToLower(requestingUserRole)

	if normalizedRole == "admin" {
		return s.repo.ListAdviseesByLecturerID(ctx, targetLecturerID)
	}

	lecturer, err := s.repo.FindLecturerByUserID(ctx, requestingUserID)
	if err != nil {
		return nil, errors.New("lecturer data not found for this user")
	}

	isOwner := lecturer.ID == targetLecturerID
	if !isOwner {
		return nil, errors.New("unauthorized: access denied to view other lecturers' advisees")
	}

	advisees, err := s.repo.ListAdviseesByLecturerID(ctx, lecturer.ID)
	if err != nil {
		return nil, err
	}

	if len(advisees) == 0 {
		return nil, errors.New("no advisees found for this lecturer")
	}

	return advisees, nil
}