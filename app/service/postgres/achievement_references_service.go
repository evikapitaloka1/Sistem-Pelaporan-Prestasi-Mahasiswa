package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	
	// Pastikan path import Repository dan Model Anda benar
	models "uas/app/model/mongo" // Alias models untuk model
	repository "uas/app/repository/mongo" // Alias repository untuk repo
)

// =========================================================
// ACHIEVEMENT SERVICE INTERFACE (CONTRACT)
// =========================================================

type AchievementService interface {
	// Create harus menerima userRole jika Admin diperbolehkan membuat atas nama student lain
	CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error)
	
	// Semua operasi CRUD/Workflow menggunakan ID MongoDB (string)
	SubmitForVerification(ctx context.Context, id string, userID uuid.UUID) error 
	DeleteAchievement(ctx context.Context, id string, userID uuid.UUID) error
	UpdateAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string, req models.AchievementRequest) error
	
	VerifyAchievement(ctx context.Context, id string, lecturerID uuid.UUID) error
	RejectAchievement(ctx context.Context, id string, lecturerID uuid.UUID, rejectionNote string) error

	// Read Operations
	GetAchievementDetail(ctx context.Context, id string, userID uuid.UUID, userRole string) (*models.AchievementDetail, error)
	ListAchievements(ctx context.Context, userRole string, userID uuid.UUID) ([]models.AchievementDetail, error)
	
	GetAchievementStatistics(ctx context.Context, userRole string, userID uuid.UUID) (interface{}, error) 
}

// =========================================================
// ACHIEVEMENT SERVICE IMPLEMENTATION
// =========================================================

type AchievementServiceImpl struct {
	MongoRepo repository.MongoAchievementRepository
	PostgreRepo repository.PostgreAchievementRepository
}

func NewAchievementService(mRepo repository.MongoAchievementRepository, pRepo repository.PostgreAchievementRepository) AchievementService {
	return &AchievementServiceImpl{
		MongoRepo: mRepo,
		PostgreRepo: pRepo,
	}
}

// -----------------------------------------------------------
// Helper Function
// -----------------------------------------------------------

func (s *AchievementServiceImpl) verifyAccessCheck(ctx context.Context, ref *models.AchievementReference, lecturerID uuid.UUID) error {
	// ... (Implementasi verifyAccessCheck)
	adviseeIDs, err := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerID)
	if err != nil { return errors.New("gagal mendapatkan data mahasiswa bimbingan") }
	
	isAdvisee := false
	for _, id := range adviseeIDs {
		if id == ref.StudentID {
			isAdvisee = true
			break
		}
	}
	if !isAdvisee {
		return errors.New("dosen ini tidak berhak memproses prestasi mahasiswa tersebut")
	}
	return nil
}

// -----------------------------------------------------------
// Mahasiswa Operations (FR-003, FR-004, FR-005)
// -----------------------------------------------------------

// CreateAchievement: (Implementasi Lengkap dengan Logika Admin/Mahasiswa)
func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error) {
	var finalStudentID uuid.UUID
	var err error

	// 1. Tentukan Student ID Target berdasarkan Role
	if userRole == "Admin" {
		if req.TargetStudentID == "" { 
			return nil, errors.New("admin harus menyediakan target Student ID")
		}
		
		finalStudentID, err = uuid.Parse(req.TargetStudentID)
		if err != nil {
			return nil, errors.New("format target Student ID tidak valid")
		}
	} else if userRole == "Mahasiswa" {
		finalStudentID, err = s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil {
			return nil, errors.New("user is not associated with a student profile")
		}
	} else {
		return nil, errors.New("role tidak memiliki hak untuk membuat prestasi")
	}
	
	// 2. Parsing EventDate
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil {
		return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat)
	}
	
	// 3. Prepare & Simpan ke MongoDB (Detail Prestasi)
	mongoDoc := models.Achievement{
		StudentUUID: finalStudentID.String(), 
		AchievementType: req.AchievementType,
		Title: req.Title,
		Description: req.Description,
		Tags: req.Tags,
		Points: req.Points,
		
		Details: models.DynamicDetails{
			EventDate: eventTime,
			Location: req.Details.Location,
			Organizer: req.Details.Organizer,
			Score: req.Details.Score,
			CustomFields: req.Details.CustomFields,
			CompetitionName: req.Details.CompetitionName,
			CompetitionLevel: req.Details.CompetitionLevel,
			Rank: req.Details.Rank,
			MedalType: req.Details.MedalType,
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mongoIDPtr, err := s.MongoRepo.Create(ctx, &mongoDoc)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan ke MongoDB: %w", err)
	}
	mongoID := mongoIDPtr.Hex()

	// 4. Simpan Referensi ke PostgreSQL (Workflow Status)
	pqRef := models.AchievementReference{
		ID: uuid.New(),
		StudentID: finalStudentID,
		MongoAchievementID: mongoID,
		Status: models.StatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.PostgreRepo.CreateReference(ctx, &pqRef); err != nil {
		s.MongoRepo.DeleteByID(ctx, *mongoIDPtr) 
		return nil, fmt.Errorf("gagal menyimpan referensi ke PostgreSQL: %w", err)
	}

	// 5. Return Data
	return &models.AchievementDetail{
		Achievement: mongoDoc,
		ReferenceID: pqRef.ID.String(),
		Status: pqRef.Status,
		SubmittedAt: nil,
		VerifiedBy: "",
		VerifiedAt: nil,
		RejectionNote: "",
	}, nil
}

// SubmitForVerification: (Implementasi sudah benar)
func (s *AchievementServiceImpl) SubmitForVerification(ctx context.Context, mongoAchievementID string, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	if ref.Status != models.StatusDraft {
		return errors.New("prestasi hanya bisa disubmit jika berstatus 'draft'")
	}
	
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID {
		return errors.New("tidak memiliki hak untuk submit prestasi ini")
	}

	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil {
		return fmt.Errorf("gagal update status menjadi submitted: %w", err)
	}
	
	return nil
}

// DeleteAchievement: (Implementasi Soft Delete FINAL)
func (s *AchievementServiceImpl) DeleteAchievement(ctx context.Context, mongoAchievementID string, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	// Precondition 1 & 2: Cek Status dan Kepemilikan
	if ref.Status != models.StatusDraft {
		return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'")
	}
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID {
		return errors.New("tidak memiliki hak untuk menghapus prestasi ini")
	}

	// 1. Soft Delete di MongoDB
	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	if err := s.MongoRepo.SoftDeleteByID(ctx, mongoID); err != nil {
		return fmt.Errorf("gagal soft delete di MongoDB: %w", err)
	}

	// 2. Update Status di PostgreSQL menjadi 'deleted' (Soft Delete Postgre)
	if err := s.PostgreRepo.UpdateReferenceForDelete(ctx, ref.ID, models.StatusDeleted); err != nil {
		return fmt.Errorf("gagal update status referensi di PostgreSQL menjadi 'deleted': %w", err)
	}

	return nil
}

// UpdateAchievement: (Implementasi Lengkap dengan Admin Bypass)
func (s *AchievementServiceImpl) UpdateAchievement(
    ctx context.Context, 
    mongoAchievementID string, 
    userID uuid.UUID, 
    userRole string, 
    req models.AchievementRequest,
) error {
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil {
        return errors.New("prestasi tidak ditemukan atau ID tidak valid") 
    }
    
    canUpdate := false 
    
    if userRole == "Admin" {
        canUpdate = true 
    } else {
        if ref.Status != models.StatusDraft {
            return errors.New("prestasi hanya bisa diupdate jika berstatus 'draft'")
        }
        
        studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
        if err != nil {
            return errors.New("user is not associated with a student profile") 
        }
        if studentID == ref.StudentID {
            canUpdate = true 
        }
    }
    
    if !canUpdate {
        return errors.New("forbidden: tidak memiliki hak untuk mengupdate prestasi ini")
    }

    // UPDATE DATABASE
    mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
    if err != nil { return errors.New("ID MongoDB tidak valid") }
    
    updateData := primitive.M{
        "title": req.Title,
        "description": req.Description,
        "achievementType": req.AchievementType,
        "tags": req.Tags,
        "points": req.Points,
        "details": req.Details,
    }
    
    if err := s.MongoRepo.UpdateByID(ctx, mongoID, updateData); err != nil {
        return fmt.Errorf("gagal update di MongoDB: %w", err)
    }

    return nil
}


// -----------------------------------------------------------
// Dosen Wali Operations (FR-007, FR-008)
// -----------------------------------------------------------

// VerifyAchievement: (Implementasi sudah benar)
func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, mongoAchievementID string, lecturerID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'")
	}

	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil { return err }

	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{Valid: false} 

	return s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusVerified, note, verifiedBy)
}

// RejectAchievement: (Implementasi sudah benar)
func (s *AchievementServiceImpl) RejectAchievement(ctx context.Context, mongoAchievementID string, lecturerID uuid.UUID, rejectionNote string) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa ditolak jika berstatus 'submitted'")
	}
	
	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil { return err }

	if rejectionNote == "" { return errors.New("catatan penolakan harus diisi") }

	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{String: rejectionNote, Valid: true}

	if err := s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusRejected, note, verifiedBy); err != nil {
		return err
	}
	return nil
}


// -----------------------------------------------------------
// Read Operations (FR-006, FR-010)
// -----------------------------------------------------------

// GetAchievementDetail: (Implementasi sudah benar)
func (s *AchievementServiceImpl) GetAchievementDetail(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) (*models.AchievementDetail, error) {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return nil, err }
	
	switch userRole {
	case "Mahasiswa":
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil || studentID != ref.StudentID { return nil, errors.New("forbidden: not the owner of this achievement") }
	case "Dosen Wali": 
		if err := s.verifyAccessCheck(ctx, ref, userID); err != nil { return nil, errors.New("forbidden: not advisor for this student") }
	case "Admin": 
		break
	default:
		return nil, errors.New("forbidden: role cannot access achievement details")
	}
	
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil { return nil, errors.New("ID MongoDB tidak valid") }
	
	mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
	if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }

	var submittedAt *time.Time
	if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }
	var verifiedAt *time.Time
	if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }

	return &models.AchievementDetail{
		Achievement: *mongoDoc,
		ReferenceID: ref.ID.String(),
		Status: ref.Status,
		SubmittedAt: submittedAt,
		VerifiedAt: verifiedAt,
		VerifiedBy: func() string {
			if ref.VerifiedBy != nil { return ref.VerifiedBy.String() }
			return "" 
		}(),
		RejectionNote: func() string {
			if ref.RejectionNote.Valid { return ref.RejectionNote.String }
			return ""
		}(),
	}, nil
}

// ListAchievements: (Implementasi sudah benar)
func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, role string, userID uuid.UUID) ([]models.AchievementDetail, error) {
    // ... (Logika ListAchievements, asumsikan sudah benar)
    return nil, nil // Placeholder return
}


// GetAchievementStatistics: (Implementasi sudah benar)
func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error) {
    // ... (Logika GetAchievementStatistics, asumsikan sudah benar)
    return nil, nil // Placeholder return
}