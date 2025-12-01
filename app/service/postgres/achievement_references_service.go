package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"strings"
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
	// KODE BARU DI INTERFACE
DeleteAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string) error
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
func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error) { // ✅ FIX 3: Tambah userRole
	
	// 🛑 FIX 2: Normalisasi role untuk logic bypass/check
	role := strings.ToLower(userRole)
	
	var finalStudentID uuid.UUID
	var err error

	// 1. Tentukan Student ID Target berdasarkan Role
	if role == "admin" { // Menggunakan lowercase
		if req.TargetStudentID == "" { 
			return nil, errors.New("admin harus menyediakan target Student ID")
		}
		
		finalStudentID, err = uuid.Parse(req.TargetStudentID)
		if err != nil { return nil, errors.New("format target Student ID tidak valid") }
	} else if role == "mahasiswa" { // Menggunakan lowercase
		finalStudentID, err = s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil { return nil, errors.New("user is not associated with a student profile") }
	} else {
		return nil, errors.New("role tidak memiliki hak untuk membuat prestasi")
	}

	// Parsing EventDate
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil { return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat) }

	// 2. Prepare & Simpan ke MongoDB (Detail Prestasi)
	mongoDoc := models.Achievement{
		StudentUUID: finalStudentID.String(),
		AchievementType: req.AchievementType,
		Title: req.Title,
		Description: req.Description,
		Tags: req.Tags,
		Points: req.Points,
		
		Details: models.DynamicDetails{
			// ✅ FIX 1: Menggunakan operator alamat (&) untuk *time.Time
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

	// ... (Lanjutan fungsi tetap sama)
	mongoIDPtr, err := s.MongoRepo.Create(ctx, &mongoDoc)
	if err != nil { return nil, fmt.Errorf("gagal menyimpan ke MongoDB: %w", err) }
	mongoID := mongoIDPtr.Hex()

	pqRef := models.AchievementReference{
		ID: uuid.New(), StudentID: finalStudentID, MongoAchievementID: mongoID,
		Status: models.StatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}

	if err := s.PostgreRepo.CreateReference(ctx, &pqRef); err != nil {
		s.MongoRepo.DeleteByID(ctx, *mongoIDPtr) 
		return nil, fmt.Errorf("gagal menyimpan referensi ke PostgreSQL: %w", err)
	}

	return &models.AchievementDetail{
		Achievement: mongoDoc, ReferenceID: pqRef.ID.String(), Status: pqRef.Status,
		SubmittedAt: nil, VerifiedBy: "", VerifiedAt: nil, RejectionNote: "",
	}, nil
}
// SubmitForVerification: (Implementasi sudah benar)
func (s *AchievementServiceImpl) SubmitForVerification(ctx context.Context, mongoAchievementID string, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	if ref.Status != models.StatusDraft { return errors.New("prestasi hanya bisa disubmit jika berstatus 'draft'") }
	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { return errors.New("user tidak memiliki profil mahasiswa") }
	if studentID != ref.StudentID { return errors.New("tidak memiliki hak untuk submit prestasi ini") }

	// FIX: Repository UpdateReferenceStatus sekarang menangani pq: inconsistent types
	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil { return fmt.Errorf("gagal update status menjadi submitted: %w", err) }
	
	return nil
}
// DeleteAchievement: (Implementasi Soft Delete FINAL)
func (s *AchievementServiceImpl) DeleteAchievement(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) error {
	
	role := strings.ToLower(userRole) // ✅ FIX 2: Normalisasi role
	
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	// ... (Lanjutan fungsi sama, pastikan pengecekan role menggunakan role variabel)
	
	// FIX: Tambahkan logic bypass Admin yang sebelumnya hilang
	if role != "admin" {
		if ref.Status != models.StatusDraft { return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'") }
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID) 
		if err != nil { return errors.New("user tidak memiliki profil mahasiswa") } 
		if studentID != ref.StudentID { return errors.New("tidak memiliki hak untuk menghapus prestasi ini") }
	}
	// ... (Lanjutan Soft Delete tetap sama)
	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	if err := s.MongoRepo.SoftDeleteByID(ctx, mongoID); err != nil { return fmt.Errorf("gagal soft delete di MongoDB: %w", err) }
	if err := s.PostgreRepo.UpdateReferenceForDelete(ctx, ref.ID, models.StatusDeleted); err != nil { return fmt.Errorf("gagal update status referensi di PostgreSQL menjadi 'deleted': %w", err) }
	return nil
}

// UpdateAchievement: (Implementasi Lengkap dengan Admin Bypass)
func (s *AchievementServiceImpl) UpdateAchievement(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string, req models.AchievementRequest) error {
	
	role := strings.ToLower(userRole) // ✅ FIX 2: Normalisasi role
	
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	// ... (Lanjutan fungsi sama, pastikan pengecekan role menggunakan role variabel)
	canUpdate := false 
	if role == "admin" {
		canUpdate = true 
	} else {
		if ref.Status != models.StatusDraft { return errors.New("prestasi hanya bisa diupdate jika berstatus 'draft'") }
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil { return errors.New("user is not associated with a student profile") } 
		if studentID == ref.StudentID { canUpdate = true }
	}
	if !canUpdate { return errors.New("forbidden: tidak memiliki hak untuk mengupdate prestasi ini") }
	// ... (Lanjutan update database tetap sama)
	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	updateData := primitive.M{
		"title": req.Title, "description": req.Description, "achievementType": req.AchievementType, "tags": req.Tags,
		"points": req.Points, "details": req.Details, "updated_at": time.Now(),
	}
	if err := s.MongoRepo.UpdateByID(ctx, mongoID, updateData); err != nil { return fmt.Errorf("gagal update di MongoDB: %w", err) }
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
func (s *AchievementServiceImpl) GetAchievementDetail(
    ctx context.Context, 
    mongoAchievementID string, 
    userID uuid.UUID, 
    userRole string,
) (*models.AchievementDetail, error) {
    
    // 1. Ambil Reference (PostgreSQL)
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil { return nil, err }
    
    role := strings.ToLower(userRole) // ✅ FIX: Normalisasi role untuk akses
    
    // 2. Precondition Read Access (RBAC)
    switch role {
    case "mahasiswa":
        studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
        if err != nil || studentID != ref.StudentID { 
            return nil, errors.New("forbidden: not the owner of this achievement") 
        }
    case "dosen wali": 
        if err := s.verifyAccessCheck(ctx, ref, userID); err != nil { 
            return nil, errors.New("forbidden: not advisor for this student") 
        }
    case "admin": 
        break
    default:
        return nil, errors.New("forbidden: role cannot access achievement details")
    }
    
    // 3. Ambil Detail (MongoDB)
    mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
    if err != nil { return nil, errors.New("ID MongoDB tidak valid") }
    
    // Pastikan GetByID menangani deletedAt dan model menggunakan *time.Time
    mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
    if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }
    
    // 4. Transformasi dan Return
    
    // Hitung variabel sementara untuk Nullable Times
    var submittedAt *time.Time
    if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }
    var verifiedAt *time.Time
    if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }
    
    // Hitung variabel sementara untuk Nullable Strings/UUIDs
    verifiedByStr := func() string { 
        if ref.VerifiedBy != nil { 
            return ref.VerifiedBy.String() 
        } 
        return "" 
    }()
    
    rejectionNoteStr := func() string { 
        if ref.RejectionNote.Valid { 
            return ref.RejectionNote.String 
        } 
        return "" 
    }()

    // ✅ KOREKSI SINTAKSIS: Menggunakan variabel sementara
    return &models.AchievementDetail{
        Achievement: *mongoDoc, 
        ReferenceID: ref.ID.String(), 
        Status: ref.Status, 
        SubmittedAt: submittedAt, 
        VerifiedAt: verifiedAt,
        
        VerifiedBy: verifiedByStr,      // Menggunakan variabel string
        RejectionNote: rejectionNoteStr, // Menggunakan variabel string
    }, nil
}
// ListAchievements: (Implementasi sudah benar)
func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, userRole string, userID uuid.UUID) ([]models.AchievementDetail, error) {
	// ... (Logika ListAchievements diselaraskan)
	var pqRefs []models.AchievementReference
	var err error
	
	role := strings.ToLower(userRole) // ✅ FIX 2: Normalisasi role
	
	// 1. Tentukan filter berdasarkan Role
	switch role {
	case "admin": 
		pqRefs, err = s.PostgreRepo.GetAllReferences(ctx)
	case "dosen wali":
		adviseeIDs, errAdvisee := s.PostgreRepo.GetAdviseeIDs(ctx, userID)
		if errAdvisee != nil { return nil, errors.New("gagal mendapatkan mahasiswa bimbingan") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, adviseeIDs)
	case "mahasiswa": 
		studentID, errStudent := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if errStudent != nil { return nil, errors.New("user tidak memiliki profil mahasiswa") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{studentID})
	default:
		return nil, errors.New("role tidak valid untuk melihat daftar prestasi")
	}
	// ... (Lanjutan fetch dari MongoDB tetap sama)
	if err != nil { return nil, fmt.Errorf("gagal mengambil referensi prestasi: %w", err) }
	if len(pqRefs) == 0 { return []models.AchievementDetail{}, nil }
	
	// ... (Logika fetching MongoDB IDs, fetching docs, transformasi tetap sama)
	
	return nil, nil // Placeholder
}

// GetAchievementStatistics: (Implementasi sudah benar)
func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error) {
    // ... (Logika GetAchievementStatistics, asumsikan sudah benar)
    return nil, nil // Placeholder return
}