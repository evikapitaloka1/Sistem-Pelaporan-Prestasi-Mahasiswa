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
	CreateAchievement(ctx context.Context, userID uuid.UUID, req models.AchievementRequest) (*models.AchievementDetail, error) 
	
	GetAchievementDetail(ctx context.Context, id string, userID uuid.UUID, role string) (*models.AchievementDetail, error)
	ListAchievements(ctx context.Context, role string, userID uuid.UUID) ([]models.AchievementDetail, error) 
	
	// ID MongoDB: Menggunakan string
	DeleteAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string) error
	
	// ID MongoDB: Menggunakan string untuk Submit
	SubmitForVerification(ctx context.Context, id string, userID uuid.UUID) error 
	
	VerifyAchievement(ctx context.Context, id string, lecturerID uuid.UUID) error
	RejectAchievement(ctx context.Context, id string, lecturerID uuid.UUID, note string) error
	
	GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error)

	UpdateAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string, req models.AchievementRequest) error
}

// =========================================================
// ACHIEVEMENT SERVICE IMPLEMENTATION
// =========================================================

// AchievementServiceImpl memegang instance dari kedua repository
type AchievementServiceImpl struct {
	MongoRepo repository.MongoAchievementRepository
	PostgreRepo repository.PostgreAchievementRepository
}

// NewAchievementService adalah constructor untuk AchievementServiceImpl
func NewAchievementService(mRepo repository.MongoAchievementRepository, pRepo repository.PostgreAchievementRepository) AchievementService {
	return &AchievementServiceImpl{
		MongoRepo: mRepo,
		PostgreRepo: pRepo,
	}
}

// -----------------------------------------------------------
// Mahasiswa Operations (FR-003, FR-004, FR-005)
// -----------------------------------------------------------

// CreateAchievement: (Implementasi sudah benar)
func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, req models.AchievementRequest) (*models.AchievementDetail, error) {
	// 1. Dapatkan Student ID dari User ID
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil {
		return nil, errors.New("user is not associated with a student profile")
	}

	// Parsing EventDate
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil {
		return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat)
	}

	// 2. Prepare & Simpan ke MongoDB (Detail Prestasi)
	mongoDoc := models.Achievement{
		StudentUUID: studentID.String(),
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

	// 3. Simpan Referensi ke PostgreSQL (Workflow Status)
	pqRef := models.AchievementReference{
		ID: uuid.New(),
		StudentID: studentID,
		MongoAchievementID: mongoID,
		Status: models.StatusDraft, 
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.PostgreRepo.CreateReference(ctx, &pqRef); err != nil {
		s.MongoRepo.DeleteByID(ctx, *mongoIDPtr) 
		return nil, fmt.Errorf("gagal menyimpan referensi ke PostgreSQL: %w", err)
	}

	// 4. Return Data
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

// SubmitForVerification: Mengubah status 'draft' menjadi 'submitted' (FR-004)
func (s *AchievementServiceImpl) SubmitForVerification(ctx context.Context, mongoAchievementID string, userID uuid.UUID) error {
	// 1. Ambil Reference (PostgreSQL) berdasarkan MongoDB ID
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return err
	}

	// Precondition 1: Status harus 'draft'
	if ref.Status != models.StatusDraft {
		return errors.New("prestasi hanya bisa disubmit jika berstatus 'draft'")
	}
	
	// Precondition 2: Pastikan yang submit adalah pemilik prestasi
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID {
		return errors.New("tidak memiliki hak untuk submit prestasi ini")
	}

	// Update status di PostgreSQL
	// Kita menggunakan ref.ID (UUID Postgre) untuk update
	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil {
		return fmt.Errorf("gagal update status menjadi submitted: %w", err)
	}
	
	// TODO: Create Notification untuk dosen wali (FR-004)

	return nil
}

// ✅ IMPLEMENTASI SOFT DELETE (DeleteAchievement)
// File: uas/app/service/mongo/achievement_service.go (Implementasi)

// File: uas/app/service/mongo/achievement_service.go (DeleteAchievement)
// File: uas/app/service/mongo/achievement_service.go (DeleteAchievement)

func (s *AchievementServiceImpl) DeleteAchievement(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string, // 🛑 Tambahkan kembali parameter userRole
) error {
	
	// 1. Dapatkan Reference
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	// 2. LOGIKA VALIDASI DAN BYPASS SUPERADMIN
	
	// Jika BUKAN Admin, jalankan semua validasi ketat
	if userRole != "Admin" {
		// --- VALIDASI MAHASISWA PEMILIK ---
		
		// Precondition 1: Status harus 'draft'
		if ref.Status != models.StatusDraft {
			return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'")
		}
		
		// Precondition 2: Cek Kepemilikan
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID) 
		if err != nil { 
			// User yang delete harus memiliki profil mahasiswa
			return errors.New("user tidak memiliki profil mahasiswa") 
		}
		
		// User ID harus sesuai dengan pemilik Achievement
		if studentID != ref.StudentID {
			return errors.New("tidak memiliki hak untuk menghapus prestasi ini")
		}
	} 
	// Jika userRole == "Admin", logic akan melewati (bypass) blok if di atas.

	// 3. LAKUKAN SOFT DELETE
	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	
	if err := s.MongoRepo.SoftDeleteByID(ctx, mongoID); err != nil {
		return fmt.Errorf("gagal soft delete di MongoDB: %w", err)
	}

	// Update Status di PostgreSQL menjadi 'deleted'
	if err := s.PostgreRepo.UpdateReferenceForDelete(ctx, ref.ID, models.StatusDeleted); err != nil {
		return fmt.Errorf("gagal update status referensi di PostgreSQL menjadi 'deleted': %w", err)
	}

	return nil
}
// Catatan: Logika bypass yang sama telah diterapkan di UpdateAchievement.
// ✅ IMPLEMENTASI UPDATE ACHIEVEMENT
func (s *AchievementServiceImpl) UpdateAchievement(
    ctx context.Context, 
    mongoAchievementID string, 
    userID uuid.UUID, 
    userRole string, 
    req models.AchievementRequest,
) error {
    
    // 1. Dapatkan Reference (Postgre) untuk cek status dan pemilik
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil {
        return errors.New("prestasi tidak ditemukan atau ID tidak valid") 
    }
    
    // 2. LOGIKA VALIDASI
    canUpdate := false 
    
    if userRole == "Admin" {
        canUpdate = true // Admin selalu diizinkan
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

    // 3. UPDATE DATABASE
    mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
    if err != nil {
        return errors.New("ID MongoDB tidak valid")
    }
    
    // Siapkan Data Update
    updateData := primitive.M{
        "title": req.Title,
        "description": req.Description,
        "achievementType": req.AchievementType,
        "tags": req.Tags,
        "points": req.Points,
        "details": req.Details,
        "updated_at": time.Now(),
    }
    
    // Panggil Mongo Repository
    if err := s.MongoRepo.UpdateByID(ctx, mongoID, updateData); err != nil {
        return fmt.Errorf("gagal update di MongoDB: %w", err)
    }

    return nil
}

// -----------------------------------------------------------
// Dosen Wali Operations (FR-007, FR-008)
// -----------------------------------------------------------

// verifyAccessCheck: Memastikan dosen adalah advisor dari mahasiswa pemilik prestasi.
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

// VerifyAchievement: Mengubah status 'submitted' menjadi 'verified' (FR-007)
func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, mongoAchievementID string, lecturerID uuid.UUID) error {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return err
	}

	// Precondition 1: Status harus 'submitted'
	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'")
	}

	// Precondition 2: Hak Akses Dosen Wali
	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil {
		return err
	}

	// Set data verifikasi
	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{Valid: false} 

	// Update status di PostgreSQL
	return s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusVerified, note, verifiedBy)
}

// RejectAchievement: Mengubah status 'submitted' menjadi 'rejected' (FR-008)
func (s *AchievementServiceImpl) RejectAchievement(ctx context.Context, mongoAchievementID string, lecturerID uuid.UUID, rejectionNote string) error {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return err
	}

	// Precondition 1: Status harus 'submitted'
	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa ditolak jika berstatus 'submitted'")
	}
	
	// Precondition 2: Hak Akses Dosen Wali
	if err := s.verifyAccessCheck(ctx, ref, lecturerID); err != nil {
		return err
	}

	if rejectionNote == "" {
		return errors.New("catatan penolakan harus diisi")
	}

	// Set data verifikasi dan catatan penolakan
	verifiedBy := sql.NullString{String: lecturerID.String(), Valid: true}
	note := sql.NullString{String: rejectionNote, Valid: true}

	// Update status di PostgreSQL
	if err := s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusRejected, note, verifiedBy); err != nil {
		return err
	}
	
	// TODO: Create Notification untuk mahasiswa (FR-008)

	return nil
}


// -----------------------------------------------------------
// Read Operations (FR-006, FR-010)
// -----------------------------------------------------------

// GetAchievementDetail: Mengambil detail lengkap dari kedua DB
func (s *AchievementServiceImpl) GetAchievementDetail(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) (*models.AchievementDetail, error) {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return nil, err
	}
	
	// 2. Precondition Read Access (RBAC):
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
		break // Admin: Full access
	default:
		return nil, errors.New("forbidden: role cannot access achievement details")
	}
	
	// 3. Ambil Detail (MongoDB)
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil {
		return nil, errors.New("ID MongoDB tidak valid")
	}
	
	// Pastikan GetByID di repo Anda memfilter field deletedAt != nil
	mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
	}

	// 4. Transformasi dan Return
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

// ListAchievements: Mengambil daftar prestasi dengan filtering berdasarkan role (FR-006, FR-010)
func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, role string, userID uuid.UUID) ([]models.AchievementDetail, error) {
	var pqRefs []models.AchievementReference
	var err error

	// 1. Tentukan filter berdasarkan Role
	switch role {
	case "Admin": // View All (FR-010)
		pqRefs, err = s.PostgreRepo.GetAllReferences(ctx)
	case "Dosen Wali": // View Mahasiswa Bimbingan (FR-006)
		adviseeIDs, errAdvisee := s.PostgreRepo.GetAdviseeIDs(ctx, userID)
		if errAdvisee != nil { return nil, errors.New("gagal mendapatkan mahasiswa bimbingan") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, adviseeIDs)
	case "Mahasiswa": // View Prestasi Sendiri
		studentID, errStudent := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if errStudent != nil { return nil, errors.New("user tidak memiliki profil mahasiswa") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{studentID})
	default:
		return nil, errors.New("role tidak valid untuk melihat daftar prestasi")
	}
	
	if err != nil { return nil, fmt.Errorf("gagal mengambil referensi prestasi: %w", err) }
	if len(pqRefs) == 0 { return []models.AchievementDetail{}, nil }

	// 2. Kumpulkan MongoDB IDs
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


// GetAchievementStatistics: Implementasi dasar untuk FR-011
func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error) {
	// TODO: Implementasi query agregasi di Repository
	return map[string]string{
		"message": "Endpoint statistik masih dalam pengembangan (FR-011). Perlu implementasi agregasi di layer Repository/Database.",
	}, nil
}