package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"strings" // ✅ WAJIB: Untuk strings.ToLower()

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
	CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error)
	
	GetAchievementDetail(ctx context.Context, id string, userID uuid.UUID, role string) (*models.AchievementDetail, error)
	ListAchievements(ctx context.Context, role string, userID uuid.UUID) ([]models.AchievementDetail, error) 
	ListAchievementsByStudentID(ctx context.Context, targetStudentID string, userID uuid.UUID, userRole string) ([]models.AchievementDetail, error)
	DeleteAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string) error
	
	SubmitForVerification(ctx context.Context, id string, userID uuid.UUID) error 
	
	VerifyAchievement(ctx context.Context, id string, lecturerID uuid.UUID) error
	RejectAchievement(ctx context.Context, id string, lecturerID uuid.UUID, note string) error
	
	GetAchievementStatistics(ctx context.Context, role string, userID uuid.UUID) (interface{}, error)
	AddAttachment(ctx context.Context, id string, userID uuid.UUID, attachment models.Attachment) error
	UpdateAchievement(ctx context.Context, id string, userID uuid.UUID, userRole string, req models.AchievementRequest) error
	GetAchievementHistory(ctx context.Context, id string, userID uuid.UUID, userRole string) ([]models.AchievementReference, error)
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
// Helper Function
// -----------------------------------------------------------

// verifyAccessCheck: Memastikan dosen adalah advisor dari mahasiswa pemilik prestasi.
// uas/app/service/mongo/achievement_service.go (Helper Function)

func (s *AchievementServiceImpl) verifyAccessCheck(ctx context.Context, ref *models.AchievementReference, userIDFromToken uuid.UUID) error {
    
    // 🛑 LANGKAH 1: KONVERSI ID TOKEN (users.id) ke ID PROFIL DOSEN (lecturers.id)
    lecturerProfileID, err := s.PostgreRepo.GetLecturerProfileID(ctx, userIDFromToken)
    if err != nil {
        // Jika user ID ada di token tapi tidak ada profil dosen, tolak.
        return errors.New("gagal mendapatkan profil dosen yang login") 
    }

    // 🛑 LANGKAH 2: Gunakan ID Profil Dosen (lecturers.id) untuk mendapatkan adviseeIDs
    adviseeIDs, err := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerProfileID) // ✅ Menggunakan ID Profil Dosen
    if err != nil { 
        return errors.New("gagal mendapatkan data mahasiswa bimbingan") 
    }
    
    // 3. Pengecekan Kepemilikan Mahasiswa
    isAdvisee := false
    // ... (Logika perulangan tetap sama)
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
// mahasiswa Operations (FR-003, FR-004, FR-005)
// -----------------------------------------------------------

// CreateAchievement: Menggunakan Pointer untuk EventDate (Fix Time Decode Error)
// File: uas/app/service/mongo/achievement_service.go (CreateAchievement)

func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error) {
	
	// Deklarasikan variabel di scope fungsi luar
	var finalStudentID uuid.UUID
	var err error

	// 1. Tentukan Student ID Target berdasarkan Role
	role := strings.ToLower(userRole) // Normalisasi

	if role == "admin" {
		if req.TargetStudentID == "" { 
			return nil, errors.New("admin harus menyediakan target Student ID")
		}
		
		finalStudentID, err = uuid.Parse(req.TargetStudentID) // Assignment
		if err != nil {
			return nil, errors.New("format target Student ID tidak valid")
		}
	} else if role == "mahasiswa" {
		finalStudentID, err = s.PostgreRepo.GetStudentProfileID(ctx, userID) // Assignment
		if err != nil {
			return nil, errors.New("user is not associated with a student profile")
		}
	} else {
		return nil, errors.New("role tidak memiliki hak untuk membuat prestasi")
	}
	
	// 🛑 KOREKSI LOGIKA BERIKUT INI MENGGUNAKAN finalStudentID YANG SUDAH DISETEL 🛑

	// 2. Parsing EventDate
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil {
		return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat)
	}

	// 3. Prepare & Simpan ke MongoDB (Detail Prestasi)
	mongoDoc := models.Achievement{
		// ✅ PENGGUNAAN 1: MENGISI STRUCT MONGODB
		StudentUUID: finalStudentID.String(), 
		AchievementType: req.AchievementType,
		Title: req.Title,
		Description: req.Description,
		Tags: req.Tags,
		Points: req.Points,
		
		Details: models.DynamicDetails{
			EventDate: eventTime, // Asumsi ini sudah menggunakan pointer jika perlu
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
		// ✅ PENGGUNAAN 2: MENGISI STRUCT POSTGRESQL
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
	// FIX: Repository UpdateReferenceStatus sekarang menangani pq: inconsistent types
	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil {
		return fmt.Errorf("gagal update status menjadi submitted: %w", err)
	}
	
	return nil
}

// DeleteAchievement: Implementasi Soft Delete dengan Normalisasi Role
func (s *AchievementServiceImpl) DeleteAchievement(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string, 
) error {
	// 🛑 KOREKSI: Normalisasi userRole
	role := strings.ToLower(userRole) 
	
	// 1. Dapatkan Reference
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	// 2. LOGIKA VALIDASI DAN BYPASS ADMIN
	if role != "admin" {
		if ref.Status != models.StatusDraft {
			return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'")
		}
		
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID) 
		if err != nil { return errors.New("user tidak memiliki profil mahasiswa") } 
		
		if studentID != ref.StudentID {
			return errors.New("tidak memiliki hak untuk menghapus prestasi ini")
		}
	} 

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

// UpdateAchievement: Implementasi Update dengan Normalisasi Role
func (s *AchievementServiceImpl) UpdateAchievement(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string, 
	req models.AchievementRequest,
) error {
	
	// 🛑 KOREKSI: Normalisasi userRole
	role := strings.ToLower(userRole) 

	// 1. Dapatkan Reference (Postgre) untuk cek status dan pemilik
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return errors.New("prestasi tidak ditemukan atau ID tidak valid") }
	
	// 2. LOGIKA VALIDASI
	canUpdate := false 
	
	if role == "admin" {
		canUpdate = true 
	} else {
		if ref.Status != models.StatusDraft {
			return errors.New("prestasi hanya bisa diupdate jika berstatus 'draft'")
		}
		
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil { return errors.New("user is not associated with a student profile") } 
		if studentID == ref.StudentID { canUpdate = true }
	}
	
	if !canUpdate {
		return errors.New("forbidden: tidak memiliki hak untuk mengupdate prestasi ini")
	}

	// 3. UPDATE DATABASE
	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	
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
// ... (tetap sama)

// VerifyAchievement: Mengubah status 'submitted' menjadi 'verified' (FR-007)
// File: uas/app/service/mongo/achievement_service.go (VerifyAchievement)

func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, mongoAchievementID string, lecturerIDFromToken uuid.UUID) error {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	// Precondition 1: Status harus 'submitted'
	if ref.Status != models.StatusSubmitted { return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'") }

    // 🛑 KOREKSI UTAMA: Lakukan pengecekan akses menggunakan ID Token Dosen
	if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil { 
        return err // <-- Error ini yang mengembalikan "dosen ini tidak berhak..."
    }

	// Set data verifikasi
	verifiedBy := sql.NullString{String: lecturerIDFromToken.String(), Valid: true}
	note := sql.NullString{Valid: false} 

	// Update status di PostgreSQL
	return s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusVerified, note, verifiedBy)
}
// RejectAchievement: Mengubah status 'submitted' menjadi 'rejected' (FR-008)
// File: uas/app/service/mongo/achievement_service.go (RejectAchievement)

func (s *AchievementServiceImpl) RejectAchievement(ctx context.Context, mongoAchievementID string, lecturerIDFromToken uuid.UUID, rejectionNote string) error {
	// 1. Dapatkan Referensi Prestasi (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

	// 2. Validasi Status dan Catatan
	if ref.Status != models.StatusSubmitted { return errors.New("prestasi hanya bisa ditolak jika berstatus 'submitted'") }
	if strings.TrimSpace(rejectionNote) == "" { return errors.New("catatan penolakan harus diisi") }

	// 🛑 KOREKSI: Panggil verifyAccessCheck dengan ID Token Dosen
	if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil { 
        return err // Mengembalikan error otorisasi
    }

	// Set data verifikasi
	verifiedBy := sql.NullString{String: lecturerIDFromToken.String(), Valid: true}
	note := sql.NullString{String: strings.TrimSpace(rejectionNote), Valid: true}

	// Update status di PostgreSQL
	if err := s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusRejected, note, verifiedBy); err != nil { return err }
	
	return nil
}
// -----------------------------------------------------------
// Read Operations (FR-006, FR-010)
// -----------------------------------------------------------

// GetAchievementDetail: Mengambil detail lengkap dari kedua DB
func (s *AchievementServiceImpl) GetAchievementDetail(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) (*models.AchievementDetail, error) {
	// 1. Ambil Reference (PostgreSQL)
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return nil, err }
	
	// ✅ FINAL FIX: Normalisasi userRole yang masuk
	role := strings.ToLower(userRole) 
	
	switch role { 
	case "mahasiswa": // 🛑 PENTING: HURUF KECIL
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil || studentID != ref.StudentID { return nil, errors.New("forbidden: not the owner of this achievement") }
	case "dosen wali": // 🛑 PENTING: HURUF KECIL
		if err := s.verifyAccessCheck(ctx, ref, userID); err != nil { return nil, errors.New("forbidden: not advisor for this student") }
	case "admin": // 🛑 PENTING: HURUF KECIL
		break 
	default:
		return nil, errors.New("forbidden: role cannot access achievement details")
	}
	
	// 3. Ambil Detail (MongoDB)
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil { return nil, errors.New("ID MongoDB tidak valid") }
	
	mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
	if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }

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
			if ref.VerifiedBy != nil { return ref.VerifiedBy.String() }
			return "" 
		}(),
		RejectionNote: func() string {
			if ref.RejectionNote.Valid { return ref.RejectionNote.String }
			return ""
		}(),
	}, nil
}

// ListAchievements: Mengambil daftar prestasi dengan filtering berdasarkan role (FR-006, FR-010)
func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, userRole string, userID uuid.UUID) ([]models.AchievementDetail, error) {
	
var pqRefs []models.AchievementReference
	var err error
	
	// ✅ FINAL FIX: Normalisasi role yang masuk
	role := strings.ToLower(userRole) 

	switch role {
	case "admin": // 🛑 PENTING: HURUF KECIL
		pqRefs, err = s.PostgreRepo.GetAllReferences(ctx)
	case "dosen wali": // 🛑 PENTING: HURUF KECIL
		adviseeIDs, errAdvisee := s.PostgreRepo.GetAdviseeIDs(ctx, userID)
		if errAdvisee != nil { return nil, errors.New("gagal mendapatkan mahasiswa bimbingan") }
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, adviseeIDs)
	case "mahasiswa": // 🛑 PENTING: HURUF KECIL
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
		if ref.Status == models.StatusDeleted {
			continue 
		}
		
		objectID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
		if err == nil {
			mongoIDs = append(mongoIDs, objectID)
			refMap[ref.MongoAchievementID] = ref
		}
	}

	// 3. Fetch Detail dari MongoDB
	mongoDocs, err := s.MongoRepo.GetByIDs(ctx, mongoIDs) 
	if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }
	
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
					if ref.VerifiedBy != nil { return ref.VerifiedBy.String() }
					return ""
				}(),
				RejectionNote: func() string {
					if ref.RejectionNote.Valid { return ref.RejectionNote.String }
					return ""
				}(),
			})
		}
	}
	
	return details, nil 
}

// GetAchievementStatistics: Implementasi dasar untuk FR-011
// File: uas/app/service/mongo/achievement_service.go (Bagian GetAchievementStatistics)

// ...

// GetAchievementStatistics: Mengembalikan statistik status prestasi (Draft, Submitted, Verified, Rejected)
// File: uas/app/service/mongo/achievement_service.go (Fungsi GetAchievementStatistics yang telah direvisi)

// ...

// GetAchievementStatistics: Mengembalikan statistik status prestasi (Draft, Submitted, Verified, Rejected)
func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, userRole string, userID uuid.UUID) (interface{}, error) {
    
    // 1. Validasi Otorisasi
    role := strings.ToLower(userRole)
    if role != "admin" {
        return nil, errors.New("forbidden: hanya Admin yang dapat mengakses statistik global")
    }

    // 2. Ambil SEMUA referensi prestasi dari PostgreSQL
    pqRefs, err := s.PostgreRepo.GetAllReferences(ctx)
    if err != nil { 
        return nil, fmt.Errorf("gagal mengambil referensi prestasi dari postgre: %w", err) 
    }
    
    // 🛑 LANGKAH 3: Agregasi Statistik (DIPERBAIKI)
    // Map untuk menghitung status.
    stats := map[string]int{
        // Baris 544-547: Konversi eksplisit ke string
        string(models.StatusDraft):     0, 
        string(models.StatusSubmitted): 0,
        string(models.StatusVerified):  0,
        string(models.StatusRejected):  0,
    }
    
    // Looping dan Agregasi
    for _, ref := range pqRefs {
        // Abaikan status deleted (Asumsi status "deleted" adalah string biasa, jika tidak, konversi)
        if ref.Status == "deleted" { 
            continue
        }
        
        // Baris 558-559: Konversi eksplisit ref.Status ke string saat mengakses map
        statusKey := string(ref.Status) // Konversi tipe models.AchievementStatus ke string
        
        // Pengecekan status yang valid sebelum increment
        if _, found := stats[statusKey]; found {
             stats[statusKey]++ 
        }
    }
    
    // ... (Lanjutan kode untuk total dan finalStats)
    total := len(pqRefs) 
    
    finalStats := map[string]interface{}{
        "total_references": total,
        "status_counts": stats, 
    }

    return finalStats, nil
}
func (s *AchievementServiceImpl) GetAchievementHistory(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) ([]models.AchievementReference, error) {
    
    // 1. Ambil Reference (PostgreSQL) untuk mendapatkan StudentID pemilik
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil { 
        return nil, errors.New("prestasi tidak ditemukan")
    }

    // 2. LOGIKA OTORISASI AKSES (RBAC Logic)
    role := strings.ToLower(userRole)
    canView := false

    switch role {
    case "admin":
        canView = true // Admin diizinkan melihat semua
        
    case "mahasiswa":
        studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
        if err == nil && studentID == ref.StudentID {
            canView = true // Mahasiswa hanya boleh melihat miliknya
        }
        
    case "dosen wali":
        if err := s.verifyAccessCheck(ctx, ref, userID); err == nil {
            canView = true // Dosen Wali hanya boleh melihat bimbingannya
        }
    }
    
    if !canView {
        return nil, errors.New("forbidden: tidak memiliki hak untuk melihat riwayat prestasi ini")
    }

    // 3. Panggil Repository untuk mengambil data history
    history, err := s.PostgreRepo.GetHistoryByMongoID(ctx, mongoAchievementID)
    if err != nil {
        return nil, fmt.Errorf("gagal mengambil riwayat dari database: %w", err)
    }
    
    // 4. Kembalikan data riwayat
    return history, nil
}


// uas/app/service/mongo/achievement_service.go

// uas/app/service/mongo/achievement_service.go

func (s *AchievementServiceImpl) AddAttachment(ctx context.Context, mongoAchievementID string, userID uuid.UUID, attachment models.Attachment) error {
    
    // 1. Ambil Reference (PostgreSQL) untuk validasi kepemilikan dan status
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil {
        return errors.New("prestasi tidak ditemukan atau ID tidak valid")
    }

    // 2. Validasi Kepemilikan (Hanya pemilik yang bisa menambah attachment)
    studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
    if err != nil { 
        return errors.New("user tidak memiliki profil mahasiswa") 
    }
    if studentID != ref.StudentID {
        return errors.New("forbidden: tidak memiliki hak untuk menambah attachment pada prestasi ini")
    }

    // 3. Validasi Status (Hanya boleh saat status DRAFT)
    if ref.Status != models.StatusDraft {
        return errors.New("attachment hanya bisa ditambahkan saat prestasi berstatus 'draft'")
    }

    // 4. Konversi ID MongoDB
    mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
    if err != nil {
        return errors.New("ID MongoDB tidak valid")
    }
    
    // 5. Panggil Mongo Repository
    // ✅ Mengatasi error type mismatch: mengirim pointer (&attachment)
    err = s.MongoRepo.AddAttachment(ctx, mongoID, &attachment) 
    if err != nil {
        // Gabungkan error dari repository dan sertakan ID prestasi
        return fmt.Errorf("gagal menyimpan attachment di MongoDB: %s", err.Error())
    }
    
    return nil
}
func (s *AchievementServiceImpl) ListAchievementsByStudentID(
    ctx context.Context, 
    targetStudentIDString string, 
    userID uuid.UUID, 
    userRole string,
) ([]models.AchievementDetail, error) {

    // 1. Validasi Student ID Target
    targetStudentID, err := uuid.Parse(targetStudentIDString)
    if err != nil {
        return nil, errors.New("format student ID tidak valid")
    }

    // 2. LOGIKA OTORISASI (RBAC/Kepemilikan)
    canView := false
    role := strings.ToLower(userRole)

    switch role {
    case "admin":
        canView = true
    case "mahasiswa":
        // Mahasiswa hanya bisa melihat achievement miliknya sendiri
        currentStudentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
        if err == nil && currentStudentID == targetStudentID {
            canView = true
        }
    case "dosen wali":
        // Dosen wali diizinkan jika ID Target adalah mahasiswa bimbingannya
        tempRef := &models.AchievementReference{StudentID: targetStudentID}
        if err := s.verifyAccessCheck(ctx, tempRef, userID); err == nil {
            canView = true
        }
    }

    if !canView {
        return nil, errors.New("forbidden: tidak memiliki hak akses untuk melihat prestasi mahasiswa ini")
    }

    // 3. Ambil References dari PostgreSQL (filter berdasarkan Student ID target)
    pqRefs, err := s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{targetStudentID})
    if err != nil {
        return nil, fmt.Errorf("gagal mengambil referensi prestasi dari postgre: %w", err)
    }

    if len(pqRefs) == 0 {
        return []models.AchievementDetail{}, nil
    }

    // 4. Proses dan filter references yang valid (tidak dihapus)
    mongoIDs := make([]primitive.ObjectID, 0, len(pqRefs))
    refMap := make(map[string]models.AchievementReference)

    for _, ref := range pqRefs {
        if ref.Status == models.StatusDeleted || ref.MongoAchievementID == "" {
            continue
        }

        objectID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
        if err != nil {
            continue // abaikan ID Mongo yang invalid
        }

        mongoIDs = append(mongoIDs, objectID)
        refMap[ref.MongoAchievementID] = ref
    }

    if len(mongoIDs) == 0 {
        return []models.AchievementDetail{}, nil
    }

    // 5. Ambil detail dari MongoDB
    mongoDocs, err := s.MongoRepo.GetByIDs(ctx, mongoIDs)
    if err != nil {
        return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
    }

    // 6. Gabungkan data Reference dan Mongo Document
    details := make([]models.AchievementDetail, 0, len(mongoDocs))
    for _, doc := range mongoDocs {
        if ref, found := refMap[doc.ID.Hex()]; found {
            var submittedAt *time.Time
            if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }

            var verifiedAt *time.Time
            if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }

            details = append(details, models.AchievementDetail{
                Achievement:   doc,
                ReferenceID:   ref.ID.String(),
                Status:        ref.Status,
                SubmittedAt:   submittedAt,
                VerifiedAt:    verifiedAt,
                VerifiedBy: func() string {
                    if ref.VerifiedBy != nil { return ref.VerifiedBy.String() }
                    return ""
                }(),
                RejectionNote: func() string {
                    if ref.RejectionNote.Valid { return ref.RejectionNote.String }
                    return ""
                }(),
            })
        }
    }

    return details, nil
}
