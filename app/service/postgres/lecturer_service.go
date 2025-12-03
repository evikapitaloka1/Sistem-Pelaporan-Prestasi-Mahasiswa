package service

import (
	"context"
	"errors"
	"strings" 

	"github.com/google/uuid"
    // ASUMSI: Model-model ini sudah didefinisikan di package uas/app/model/postgres
	model "uas/app/model/postgres" 
	lecturerRepo "uas/app/repository/postgres"
)

// LecturerService mendefinisikan operasi bisnis untuk Dosen.
type LecturerService interface {
    // Fungsi ini menerima ID dan Role untuk filtering (Admin/Self-Access).
	GetAllLecturers(ctx context.Context, requestingUserID uuid.UUID, requestingUserRole string) ([]model.Lecturer, error) 
	
    // Fungsi ini menerapkan otorisasi kepemilikan.
	GetAdvisees(ctx context.Context, targetLecturerID uuid.UUID, requestingUserID uuid.UUID, requestingUserRole string) ([]model.Advisee, error)
}

type lecturerSvc struct {
	repo lecturerRepo.LecturerRepository
}

// NewLecturerService adalah constructor untuk inisialisasi service
func NewLecturerService(repo lecturerRepo.LecturerRepository) LecturerService {
	return &lecturerSvc{repo: repo}
}

// =========================================================================================
// GetAllLecturers: Menerapkan Logika Self-Access (Dosen hanya lihat data sendiri)
// =========================================================================================
func (s *lecturerSvc) GetAllLecturers(
    ctx context.Context,
    requestingUserID uuid.UUID,
    requestingUserRole string,
) ([]model.Lecturer, error) {

    normalizedRole := strings.ToLower(requestingUserRole)

    // 1. --- ADMIN: Full Access ---
    if normalizedRole == "admin" {
        return s.repo.ListAllLecturers(ctx)
    }

    // 2. --- DOSEN: Self-Access ---
    isLecturer := normalizedRole == "dosen wali" || normalizedRole == "lecturer" || normalizedRole == "pengajar"

    if isLecturer {
        lecturer, err := s.repo.FindLecturerByUserID(ctx, requestingUserID)
        if err != nil {
            return nil, errors.New("lecturer data not found for this user")
        }

        return []model.Lecturer{*lecturer}, nil
    }

    // 3. --- Deny Akses Role Lain ---
    return nil, errors.New("unauthorized: only lecturers and administrators can view this list")
}



// =========================================================================================
// GetAdvisees: Menerapkan Logika Self-Access (Dosen hanya lihat mahasiswa bimbingannya)
// =========================================================================================
func (s *lecturerSvc) GetAdvisees(
    ctx context.Context, 
    targetLecturerID uuid.UUID, 
    requestingUserID uuid.UUID, 
    requestingUserRole string,
) ([]model.Advisee, error) {

    normalizedRole := strings.ToLower(requestingUserRole)

    // 1. Admin boleh akses semua lecturer
    if normalizedRole == "admin" {
        return s.repo.ListAdviseesByLecturerID(ctx, targetLecturerID)
    }

    // 2. Ambil lecturer ID milik user login
    lecturer, err := s.repo.FindLecturerByUserID(ctx, requestingUserID)
    if err != nil {
        return nil, errors.New("lecturer data not found for this user")
    }

    // Pastikan dosen hanya akses bimbingannya sendiri
    isOwner := lecturer.ID == targetLecturerID
    if !isOwner {
        return nil, errors.New("unauthorized: access denied to view other lecturers' advisees")
    }

    // 3. Akses diperbolehkan → ambil advisees milik lecturer
    advisees, err := s.repo.ListAdviseesByLecturerID(ctx, lecturer.ID)
    if err != nil {
        return nil, err
    }

    if len(advisees) == 0 {
        return nil, errors.New("no advisees found for this lecturer")
    }

    return advisees, nil
}
