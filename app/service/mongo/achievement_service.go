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

	models "uas/app/model/mongo" 
	repository "uas/app/repository/mongo" 
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
// Helper Function: Pengecekan Akses Dosen Wali
// -----------------------------------------------------------

func (s *AchievementServiceImpl) verifyAccessCheck(ctx context.Context, ref *models.AchievementReference, userIDFromToken uuid.UUID) error {
	
	// LANGKAH 1: KONVERSI ID TOKEN (users.id) ke ID PROFIL DOSEN (lecturers.id)
	lecturerProfileID, err := s.PostgreRepo.GetLecturerProfileID(ctx, userIDFromToken)
	if err != nil {
		// 🛑 Error yang dioptimalkan untuk debugging data
		return fmt.Errorf("gagal mendapatkan profil dosen yang login (User ID: %s)", userIDFromToken.String()) 
	}

	// LANGKAH 2: Gunakan ID Profil Dosen untuk mendapatkan adviseeIDs
	adviseeIDs, err := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerProfileID) 
	if err != nil { 
		return errors.New("gagal mendapatkan data mahasiswa bimbingan") 
	}
	
	// LANGKAH 3: Pengecekan Kepemilikan Mahasiswa
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
// Operasi CRUD Mahasiswa/Admin
// -----------------------------------------------------------

func (s *AchievementServiceImpl) CreateAchievement(ctx context.Context, userID uuid.UUID, userRole string, req models.AchievementRequest) (*models.AchievementDetail, error) {
	
	var finalStudentID uuid.UUID
	var err error

	role := strings.ToLower(userRole) 

	if role == "admin" {
		if req.TargetStudentID == "" { 
			return nil, errors.New("admin harus menyediakan target Student ID")
		}
		
		finalStudentID, err = uuid.Parse(req.TargetStudentID) 
		if err != nil {
			return nil, errors.New("format target Student ID tidak valid")
		}
	} else if role == "mahasiswa" {
		finalStudentID, err = s.PostgreRepo.GetStudentProfileID(ctx, userID) 
		if err != nil {
			return nil, errors.New("user is not associated with a student profile")
		}
	} else {
		return nil, errors.New("role tidak memiliki hak untuk membuat prestasi")
	}
	
	const dateFormat = "2006-01-02" 
	eventTime, err := time.Parse(dateFormat, req.Details.EventDate)
	if err != nil {
		return nil, fmt.Errorf("format eventDate tidak valid. Harap gunakan %s", dateFormat)
	}

	mongoDoc := models.Achievement{
		StudentID: finalStudentID, 
		AchievementType: req.AchievementType,
		Title: req.Title,
		Description: req.Description,
		Tags: req.Tags,
		Points: req.Points,
		
		Details: models.DynamicDetails{
			EventDate: &eventTime, 
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

func (s *AchievementServiceImpl) SubmitForVerification(ctx context.Context, mongoAchievementID string, userID uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
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

	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusSubmitted, sql.NullString{}, sql.NullString{})
	if err != nil {
		return fmt.Errorf("gagal update status menjadi submitted: %w", err)
	}
    
	return nil
}

func (s *AchievementServiceImpl) DeleteAchievement(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string, 
) error {
	role := strings.ToLower(userRole) 
	
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return err }

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

	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	
	if err := s.MongoRepo.SoftDeleteByID(ctx, mongoID); err != nil {
		return fmt.Errorf("gagal soft delete di MongoDB: %w", err)
	}

	if err := s.PostgreRepo.UpdateReferenceForDelete(ctx, ref.ID, models.StatusDeleted); err != nil {
		return fmt.Errorf("gagal update status referensi di PostgreSQL menjadi 'deleted': %w", err)
	}

	return nil
}

func (s *AchievementServiceImpl) UpdateAchievement(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string, 
	req models.AchievementRequest,
) error {
	
	role := strings.ToLower(userRole) 

	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return errors.New("prestasi tidak ditemukan atau ID tidak valid") }
	
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

	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil { return errors.New("ID MongoDB tidak valid") }
	
	updateData := primitive.M{
		"title": req.Title,
		"description": req.Description,
		"achievementType": req.AchievementType,
		"tags": req.Tags,
		"points": req.Points,
		"details": req.Details,
		"updated_at": time.Now(), 
	}
	
	if err := s.MongoRepo.UpdateByID(ctx, mongoID, updateData); err != nil {
		return fmt.Errorf("gagal update di MongoDB: %w", err)
	}

	return nil
}

// -----------------------------------------------------------
// Dosen Wali Operations
// -----------------------------------------------------------

// service/achievement_service.go

// service/achievement_service.go

func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, mongoAchievementID string, lecturerIDFromToken uuid.UUID) error {
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil { 
        return err 
    }

    if ref.Status != models.StatusSubmitted { 
        return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'") 
    }

    // ✅ Pengecekan Akses Dosen Wali (memastikan dosen ini adalah advisor mahasiswa ybs)
    // Fungsi ini secara internal memanggil GetLecturerProfileID dan GetAdviseeIDs
    if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil {
        return err
    }

    // ✅ PERBAIKAN: Gunakan langsung lecturerIDFromToken (yang merupakan users.id)
    verifiedByID := lecturerIDFromToken 
    
    // Siapkan data untuk update status
    verifiedBy := sql.NullString{String: verifiedByID.String(), Valid: true}
    note := sql.NullString{Valid: false} 

    // Panggil Repository
    err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusVerified, note, verifiedBy)
    if err != nil {
        // Jika masih FK violation, ID user dari token tidak ada di tabel users
        return fmt.Errorf("gagal update status menjadi verified: %w", err)
    }

    return nil
}

// Fungsi RejectAchievement
// service/achievement_service.go

// service/achievement_service.go

func (s *AchievementServiceImpl) RejectAchievement(ctx context.Context, mongoAchievementID string, lecturerIDFromToken uuid.UUID, rejectionNote string) error {
    ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
    if err != nil { 
        return err 
    }

    if rejectionNote == "" { 
        return errors.New("catatan penolakan harus diisi") 
    }

    if ref.Status != models.StatusSubmitted {
        return errors.New("prestasi hanya bisa ditolak jika berstatus 'submitted'")
    }
    
    // ✅ Pengecekan Akses Dosen Wali
    if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil { 
        return err 
    }
    
    // ✅ PERBAIKAN: Gunakan langsung lecturerIDFromToken (yang merupakan users.id)
    verifiedByID := lecturerIDFromToken 

    verifiedBy := sql.NullString{String: verifiedByID.String(), Valid: true} 
    note := sql.NullString{String: rejectionNote, Valid: true}

    // Panggil Repository
    err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusRejected, note, verifiedBy)
    if err != nil {
        return fmt.Errorf("gagal update status menjadi rejected: %w", err)
    }
    
    return nil
}
// -----------------------------------------------------------
// Read Operations
// -----------------------------------------------------------

func (s *AchievementServiceImpl) GetAchievementDetail(
	ctx context.Context, 
	mongoAchievementID string, 
	userID uuid.UUID, 
	userRole string,
) (*models.AchievementDetail, error) {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { return nil, err }
	
	role := strings.ToLower(userRole) 
	
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
	
	mongoID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
	if err != nil { return nil, errors.New("ID MongoDB tidak valid") }
	
	mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
	if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }
	
	var submittedAt *time.Time
	if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }
	var verifiedAt *time.Time
	if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }
	
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

	return &models.AchievementDetail{
		Achievement: *mongoDoc, 
		ReferenceID: ref.ID.String(), 
		Status: ref.Status, 
		SubmittedAt: submittedAt, 
		VerifiedAt: verifiedAt,
		VerifiedBy: verifiedByStr, 
		RejectionNote: rejectionNoteStr, 
	}, nil
}

func (s *AchievementServiceImpl) ListAchievements(ctx context.Context, userRole string, userID uuid.UUID) ([]models.AchievementDetail, error) {
	
	var pqRefs []models.AchievementReference
	var err error
	
	role := strings.ToLower(userRole) 

	switch role {
	case "admin":
		pqRefs, err = s.PostgreRepo.GetAllReferences(ctx)
	case "dosen wali":
		lecturerProfileID, errLecturer := s.PostgreRepo.GetLecturerProfileID(ctx, userID)
		if errLecturer != nil {
			// 🛑 PERBAIKAN: Mengembalikan error yang lebih informatif
			return nil, fmt.Errorf("gagal mendapatkan profil dosen yang login (User ID: %s): %w", userID.String(), errLecturer)
		}
		adviseeIDs, errAdvisee := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerProfileID)
		if errAdvisee != nil { 
			return nil, fmt.Errorf("gagal mendapatkan mahasiswa bimbingan: %w", errAdvisee) 
		}
		if len(adviseeIDs) == 0 {
			return []models.AchievementDetail{}, nil
		}
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, adviseeIDs)
	case "mahasiswa":
		studentID, errStudent := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if errStudent != nil { 
			return nil, fmt.Errorf("user tidak memiliki profil mahasiswa: %w", errStudent) 
		}
		pqRefs, err = s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{studentID})
	default:
		return nil, errors.New("role tidak valid untuk melihat daftar prestasi")
	}
	
	if err != nil { return nil, fmt.Errorf("gagal mengambil referensi prestasi dari PostgreSQL: %w", err) }
	
	if len(pqRefs) == 0 { 
		return []models.AchievementDetail{}, nil 
	}

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
	
	if len(mongoIDs) == 0 {
		return []models.AchievementDetail{}, nil
	}

	mongoDocs, err := s.MongoRepo.GetByIDs(ctx, mongoIDs) 
	if err != nil { return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err) }
	
	if len(mongoDocs) == 0 {
		return []models.AchievementDetail{}, nil
	}
	
	details := make([]models.AchievementDetail, 0, len(mongoDocs))
	for _, doc := range mongoDocs {
		ref, found := refMap[doc.ID.Hex()]
		if found {
			var submittedAt *time.Time
			if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }
			var verifiedAt *time.Time
			if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }
			
			verifiedByStr := func() string { 
				if ref.VerifiedBy != nil { return ref.VerifiedBy.String() } 
				return "" 
			}()
			
			rejectionNoteStr := func() string { 
				if ref.RejectionNote.Valid { return ref.RejectionNote.String } 
				return "" 
			}()

			details = append(details, models.AchievementDetail{
				Achievement: doc,
				ReferenceID: ref.ID.String(),
				Status: ref.Status,
				SubmittedAt: submittedAt,
				VerifiedAt: verifiedAt,
				VerifiedBy: verifiedByStr,
				RejectionNote: rejectionNoteStr,
			})
		}
	}
	
	return details, nil 
}

func (s *AchievementServiceImpl) GetAchievementStatistics(ctx context.Context, userRole string, userID uuid.UUID) (interface{}, error) {
	
	role := strings.ToLower(userRole)
	if role != "admin" {
		return nil, errors.New("forbidden: hanya Admin yang dapat mengakses statistik global")
	}

	pqRefs, err := s.PostgreRepo.GetAllReferences(ctx)
	if err != nil { 
		return nil, fmt.Errorf("gagal mengambil referensi prestasi dari postgre: %w", err) 
	}
	
	stats := map[string]int{
		string(models.StatusDraft): 0, 
		string(models.StatusSubmitted): 0,
		string(models.StatusVerified): 0,
		string(models.StatusRejected): 0,
	}
	
	for _, ref := range pqRefs {
		if ref.Status == models.StatusDeleted { 
			continue
		}
		
		statusKey := string(ref.Status) 
		
		if _, found := stats[statusKey]; found {
			stats[statusKey]++ 
		}
	}
	
	total := len(pqRefs) 
	
	finalStats := map[string]interface{}{
		"total_references": total,
		"status_counts": stats, 
	}

	return finalStats, nil
}

func (s *AchievementServiceImpl) AddAttachment(ctx context.Context, mongoAchievementID string, userID uuid.UUID, attachment models.Attachment) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return errors.New("prestasi tidak ditemukan atau ID tidak valid")
	}

	studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
	if err != nil { 
		return errors.New("user tidak memiliki profil mahasiswa") 
	}
	if studentID != ref.StudentID {
		return errors.New("forbidden: tidak memiliki hak untuk menambah attachment pada prestasi ini")
	}

	if ref.Status != models.StatusDraft {
		return errors.New("attachment hanya bisa ditambahkan saat prestasi berstatus 'draft'")
	}

	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil {
		return errors.New("ID MongoDB tidak valid")
	}
	
	err = s.MongoRepo.AddAttachment(ctx, mongoID, &attachment) 
	if err != nil {
		return fmt.Errorf("gagal menyimpan attachment di MongoDB: %s", err.Error())
	}
	
	return nil
}

func (s *AchievementServiceImpl) GetAchievementHistory(ctx context.Context, mongoAchievementID string, userID uuid.UUID, userRole string) ([]models.AchievementReference, error) {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil { 
		return nil, errors.New("prestasi tidak ditemukan")
	}

	role := strings.ToLower(userRole)
	
	switch role {
	case "mahasiswa":
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil || studentID != ref.StudentID {
			return nil, errors.New("forbidden: tidak memiliki akses ke riwayat prestasi ini")
		}
	case "dosen wali":
		if err := s.verifyAccessCheck(ctx, ref, userID); err != nil { 
			return nil, errors.New("forbidden: tidak memiliki akses ke riwayat prestasi ini") 
		}
	case "admin":
		break 
	default:
		return nil, errors.New("forbidden: role tidak dapat mengakses riwayat")
	}

	history, err := s.PostgreRepo.GetHistoryByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat dari database: %w", err)
	}
	
	return history, nil
}

func (s *AchievementServiceImpl) ListAchievementsByStudentID(
	ctx context.Context, 
	targetStudentIDString string, 
	userID uuid.UUID, 
	userRole string,
) ([]models.AchievementDetail, error) {
	var targetStudentID uuid.UUID
	if targetStudentIDString == "" {
		return nil, errors.New("target student ID tidak boleh kosong")
	}
	targetStudentID, err := uuid.Parse(targetStudentIDString)
	if err != nil {
		return nil, fmt.Errorf("target student ID tidak valid: %w", err)
	}

	canView := false 
	
	if strings.ToLower(userRole) == "admin" {
		canView = true
	} else if strings.ToLower(userRole) == "mahasiswa" {
		profileID, _ := s.PostgreRepo.GetStudentProfileID(ctx, userID) 
		if profileID == targetStudentID {
			canView = true
		}
	} else if strings.ToLower(userRole) == "dosen wali" {
		lecturerProfileID, err := s.PostgreRepo.GetLecturerProfileID(ctx, userID)
		if err == nil {
			adviseeIDs, _ := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerProfileID)
			for _, id := range adviseeIDs {
				if id == targetStudentID {
					canView = true
					break
				}
			}
		}
	}

	if !canView {
		return nil, errors.New("forbidden: tidak memiliki hak akses untuk melihat prestasi mahasiswa ini")
	}

	pqRefs, err := s.PostgreRepo.GetReferencesByStudentIDs(ctx, []uuid.UUID{targetStudentID})
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil referensi prestasi dari postgre: %w", err)
	}

	if len(pqRefs) == 0 {
		return []models.AchievementDetail{}, nil
	}

	mongoIDs := make([]primitive.ObjectID, 0, len(pqRefs))
	refMap := make(map[string]models.AchievementReference)

	for _, ref := range pqRefs {
		if ref.Status == models.StatusDeleted || ref.MongoAchievementID == "" {
			continue
		}

		objectID, err := primitive.ObjectIDFromHex(ref.MongoAchievementID)
		if err != nil {
			continue
		}

		mongoIDs = append(mongoIDs, objectID)
		refMap[ref.MongoAchievementID] = ref
	}
	
	if len(mongoIDs) == 0 {
		return []models.AchievementDetail{}, nil
	}

	mongoDocs, err := s.MongoRepo.GetByIDs(ctx, mongoIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
	}
	
	if len(mongoDocs) == 0 {
		return []models.AchievementDetail{}, nil
	}

	details := make([]models.AchievementDetail, 0, len(mongoDocs))
	for _, doc := range mongoDocs {
		ref, ok := refMap[doc.ID.Hex()]
		if !ok {
			continue
		}
		
		var submittedAt *time.Time
		if ref.SubmittedAt.Valid { submittedAt = &ref.SubmittedAt.Time }

		var verifiedAt *time.Time
		if ref.VerifiedAt.Valid { verifiedAt = &ref.VerifiedAt.Time }

		var rejectionNote string
		if ref.RejectionNote.Valid { rejectionNote = ref.RejectionNote.String }

		var verifiedBy string
		if ref.VerifiedBy != nil { verifiedBy = ref.VerifiedBy.String() }


		details = append(details, models.AchievementDetail{
			Achievement: doc, 
			ReferenceID: ref.ID.String(),
			Status: ref.Status,
			SubmittedAt: submittedAt,
			VerifiedAt: verifiedAt,
			VerifiedBy: verifiedBy,
			RejectionNote: rejectionNote,
		})
	}
	
	return details, nil
}