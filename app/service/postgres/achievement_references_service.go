package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	
	models "uas/app/model/mongo" 
	repository "uas/app/repository/postgres" // Sesuaikan path jika perlu
)

// =========================================================
// ACHIEVEMENT SERVICE INTERFACE (CONTRACT)
// =========================================================

type AchievementService interface {
    // 🎯 PERBAIKAN 1: Tambahkan userRole ke parameter CreateAchievement
	CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error)
    
	SubmitForVerification(ctx context.Context, refID uuid.UUID, userID uuid.UUID) error
	DeleteAchievement(ctx context.Context, refID uuid.UUID, userID uuid.UUID) error
	
	VerifyAchievement(ctx context.Context, refID uuid.UUID, lecturerID uuid.UUID) error
	RejectAchievement(ctx context.Context, refID uuid.UUID, lecturerID uuid.UUID, rejectionNote string) error

	GetAchievementDetail(ctx context.Context, refID uuid.UUID, userID uuid.UUID, userRole string) (*models.AchievementDetail, error)
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

// CreateAchievement: Simpan ke MongoDB (Detail) dan PostgreSQL (Reference/Status)
// File: uas/app/service/postgres/achievement_references_service.go

func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error) {
	
	var finalStudentID uuid.UUID
	var err error

	// 1. Tentukan Student ID Target berdasarkan Role
	if userRole == "Admin" {
        // Logika Admin: Harus menentukan target student ID untuk Full Access
		if req.TargetStudentID == "" { 
            return nil, errors.New("admin harus menyediakan target Student ID")
        }
		
		finalStudentID, err = uuid.Parse(req.TargetStudentID)
		if err != nil {
			return nil, errors.New("format target Student ID tidak valid")
		}
		// TODO: Lakukan pengecekan apakah studentID target benar-benar ada di tabel students (opsional)
		
	} else if userRole == "Mahasiswa" {
        // Logika Mahasiswa: Harus memiliki profil sendiri
		finalStudentID, err = s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil {
            // Mengeluarkan error jika Mahasiswa tidak memiliki profil
			return nil, errors.New("user is not associated with a student profile")
		}
	} else {
        // Tolak role lain (misalnya Dosen Wali)
		return nil, errors.New("role tidak memiliki hak untuk membuat prestasi")
	}
	
	// 2. Parsing EventDate (Menangani format YYYY-MM-DD)
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil {
		return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat)
	}
	
	// 3. Prepare & Simpan ke MongoDB (Detail Prestasi)
	
	// Catatan: Jika ada field tanggal lain di DetailsRequest (misalnya Period.Start/End),
	// Anda perlu menambahkan logika parsing serupa di sini.
	
	mongoDoc := models.Achievement{
		StudentUUID: finalStudentID.String(), // Menggunakan Student ID yang sudah ditentukan
		AchievementType: req.AchievementType,
		Title: req.Title,
		Description: req.Description,
		Tags: req.Tags,
		Points: req.Points,
		
		// Mapping Details (Mapping fields dari DTO Request ke DB Model)
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
			PublicationType: req.Details.PublicationType,
			PublicationTitle: req.Details.PublicationTitle,
			Authors: req.Details.Authors,
			Publisher: req.Details.Publisher,
			ISSN: req.Details.ISSN,
			OrganizationName: req.Details.OrganizationName,
			Position: req.Details.Position,
			// ... Lanjutkan mapping semua field details
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
		StudentID: finalStudentID, // Menggunakan Student ID yang sudah ditentukan
		MongoAchievementID: mongoID,
		Status: models.StatusDraft, // Status awal: 'draft'
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.PostgreRepo.CreateReference(ctx, &pqRef); err != nil {
		// Rollback MongoDB jika PostgreSQL gagal
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
// -----------------------------------------------------------
// Submit, Delete, Verify, Reject, Read, List, Statistics Operations
// -----------------------------------------------------------

func (s *AchievementServiceImpl) SubmitForVerification(ctx context.Context, refID uuid.UUID, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByID(ctx, refID)
	if err != nil {
		return err
	}
	
	if ref.Status != models.StatusDraft {
		return errors.New("prestasi hanya bisa disubmit jika berstatus 'draft'")
	}
	
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID {
		return errors.New("tidak memiliki hak untuk submit prestasi ini")
	}

	err = s.PostgreRepo.UpdateReferenceStatus(ctx, refID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil {
		return fmt.Errorf("gagal update status menjadi submitted: %w", err)
	}
	
	return nil
}

func (s *AchievementServiceImpl) DeleteAchievement(ctx context.Context, refID uuid.UUID, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByID(ctx, refID)
	if err != nil {
		return err
	}

	if ref.Status != models.StatusDraft {
		return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'")
	}
	
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID {
		return errors.New("tidak memiliki hak untuk menghapus prestasi ini")
	}

	// 1. Soft Delete di MongoDB
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil {
		return errors.New("ID MongoDB tidak valid")
	}
	if err := s.MongoRepo.SoftDeleteByID(ctx, mongoID); err != nil {
		return fmt.Errorf("gagal soft delete di MongoDB: %w", err)
	}

	// 2. Hard Delete Reference di PostgreSQL (Karena status pasti 'draft')
	if err := s.PostgreRepo.UpdateReferenceForDelete(ctx, refID); err != nil {
		return fmt.Errorf("gagal menghapus referensi di PostgreSQL: %w", err)
	}

	return nil
}

func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, refID uuid.UUID, lecturerID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByID(ctx, refID)
	if err != nil {
		return err
	}

	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'")
	}

	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil {
		return err
	}

	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{Valid: false} 

	return s.PostgreRepo.UpdateReferenceStatus(ctx, refID, models.StatusVerified, note, verifiedBy)
}

func (s *AchievementServiceImpl) RejectAchievement(ctx context.Context, refID uuid.UUID, lecturerID uuid.UUID, rejectionNote string) error {
	ref, err := s.PostgreRepo.GetReferenceByID(ctx, refID)
	if err != nil {
		return err
	}

	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa ditolak jika berstatus 'submitted'")
	}
	
	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil {
		return err
	}

	if rejectionNote == "" {
		return errors.New("catatan penolakan harus diisi")
	}

	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{String: rejectionNote, Valid: true}

	if err := s.PostgreRepo.UpdateReferenceStatus(ctx, refID, models.StatusRejected, note, verifiedBy); err != nil {
		return err
	}
	
	// TODO: Create Notification untuk mahasiswa

	return nil
}

func (s *AchievementServiceImpl) GetAchievementDetail(ctx context.Context, refID uuid.UUID, userID uuid.UUID, userRole string) (*models.AchievementDetail, error) {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByID(ctx, refID)
	if err != nil {
		return nil, err
	}

	// 2. Precondition Read Access (RBAC)
	switch userRole {
	case "Mahasiswa":
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil || studentID != ref.StudentID {
			return nil, errors.New("forbidden: not the owner of this achievement")
		}
	case "Dosen Wali":
		if err := s.verifyAccessCheck(ctx, ref, userID); err != nil {
			 return nil, errors.New("forbidden: not advisor for this student")
		}
	case "Admin":
		break // Admin memiliki full access
	default:
		return nil, errors.New("forbidden: role cannot access achievement details")
	}

	// 3. Ambil Detail (MongoDB)
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil {
		return nil, errors.New("ID MongoDB tidak valid")
	}
	
	mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
	}

	// 4. Transformasi dan Return (Handle Nullable fields)
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
			if ref.VerifiedBy != nil {
				return ref.VerifiedBy.String()
			}
			return ""
		}(), 
		
		RejectionNote: func() string {
			if ref.RejectionNote.Valid {
				return ref.RejectionNote.String
			}
			return ""
		}(),
	}, nil
}

func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, role string, userID uuid.UUID) ([]models.AchievementDetail, error) {
	var pqRefs []models.AchievementReference
	var err error

	// 1. Tentukan filter berdasarkan Role
	switch role {
	case "Admin": 
		pqRefs, err = s.PostgreRepo.GetAllReferences(ctx)
	case "Dosen Wali": 
		adviseeIDs, errAdvisee := s.PostgreRepo.GetAdviseeIDs(ctx, userID)
		if errAdvisee != nil { return nil, errors.New("gagal mendapatkan mahasiswa bimbingan") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, adviseeIDs)
	case "Mahasiswa": 
		studentID, errStudent := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if errStudent != nil { return nil, errors.New("user tidak memiliki profil mahasiswa") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{studentID})
	default:
		return nil, errors.New("role tidak valid untuk melihat daftar prestasi")
	}
	
	if err != nil { return nil, fmt.Errorf("gagal mengambil referensi prestasi: %w", err) }
	if len(pqRefs) == 0 { return []models.AchievementDetail{}, nil }

	// 2. Kumpulkan MongoDB IDs dan Map Referensi
	mongoIDs := make([]primitive.ObjectID, 0, len(pqRefs))
	refMap := make(map[string]models.AchievementReference) 
	for _, ref := range pqRefs {
		objectID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
		if err == nil {
			mongoIDs = append(mongoIDs, objectID)
			refMap[ref.MongoAchievementID] = ref
		}
	}

	// 3. Fetch Detail dari MongoDB
	mongoDocs, err := s.MongoRepo.GetByIDs(ctx, mongoIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
	}
	
	// 4. Gabungkan dan Transformasi
	details := make([]models.AchievementDetail, 0, len(mongoDocs))
	for _, doc := range mongoDocs {
		ref, found := refMap[doc.ID.Hex()]
		if found {
			var submittedAt *time.Time
			if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }
			var verifiedAt *time.Time
			if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }
			
			details = append(details, models.AchievementDetail{
				Achievement: doc,
				ReferenceID: ref.ID.String(),
				Status: ref.Status,
				SubmittedAt: submittedAt,
				VerifiedAt: verifiedAt,
				
				VerifiedBy: func() string {
					if ref.VerifiedBy != nil {
						return ref.VerifiedBy.String()
					}
					return ""
				}(),
				
				RejectionNote: func() string {
					if ref.RejectionNote.Valid {
						return ref.RejectionNote.String
					}
					return ""
				}(),
			})
		}
	}
	
	return details, nil
}

func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error) {
	// Implementasi ini harus didasarkan pada FR-011
	return map[string]string{
		"message": "Endpoint statistik masih dalam pengembangan (FR-011).",
		"requirement": "Statistik harus digenerate berdasarkan role (Mahasiswa: own, Dosen Wali: advisee, Admin: all)",
	}, nil
}