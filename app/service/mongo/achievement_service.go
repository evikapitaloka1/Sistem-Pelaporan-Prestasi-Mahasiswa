package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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
	if err != nil {
		return errors.New("user tidak memiliki profil mahasiswa")
	}
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
	if err != nil {
		return err
	}

	if role != "admin" {
		if ref.Status != models.StatusDraft {
			return errors.New("prestasi hanya bisa dihapus jika berstatus 'draft'")
		}

		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil {
			return errors.New("user tidak memiliki profil mahasiswa")
		}

		if studentID != ref.StudentID {
			return errors.New("tidak memiliki hak untuk menghapus prestasi ini")
		}
	}

	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil {
		return errors.New("ID MongoDB tidak valid")
	}

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
	if err != nil {
		return errors.New("prestasi tidak ditemukan atau ID tidak valid")
	}

	canUpdate := false

	if role == "admin" {
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

	mongoID, err := primitive.ObjectIDFromHex(mongoAchievementID)
	if err != nil {
		return errors.New("ID MongoDB tidak valid")
	}

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

func (s *AchievementServiceImpl) VerifyAchievement(ctx context.Context, mongoAchievementID string, lecturerIDFromToken uuid.UUID) error {
	ref, err := s.PostgreRepo.GetReferenceByMongoID(ctx, mongoAchievementID)
	if err != nil {
		return err
	}

	if ref.Status != models.StatusSubmitted {
		return errors.New("prestasi hanya bisa diverifikasi jika berstatus 'submitted'")
	}

	if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil {
		return err
	}

	verifiedByID := lecturerIDFromToken

	verifiedBy := sql.NullString{String: verifiedByID.String(), Valid: true}
	note := sql.NullString{Valid: false}

	err = s.PostgreRepo.UpdateReferenceStatus(ctx, ref.ID, models.StatusVerified, note, verifiedBy)
	if err != nil {
		return fmt.Errorf("gagal update status menjadi verified: %w", err)
	}

	return nil
}

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

	if err := s.verifyAccessCheck(ctx, ref, lecturerIDFromToken); err != nil {
		return err
	}

	verifiedByID := lecturerIDFromToken

	verifiedBy := sql.NullString{String: verifiedByID.String(), Valid: true}
	note := sql.NullString{String: rejectionNote, Valid: true}

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
    if err != nil {
        return nil, err
    }

    role := strings.ToLower(userRole)

    switch role {
    case "mahasiswa":
        studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
        if err != nil {
            fmt.Println("DEBUG: GetStudentProfileID error:", err)
            return nil, err
        }

        fmt.Println("DEBUG: JWT userID:", userID)
        fmt.Println("DEBUG: StudentID from repo:", studentID)
        fmt.Println("DEBUG: StudentID from achievement ref:", ref.StudentID)

        if studentID != ref.StudentID {
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
    if err != nil {
        return nil, errors.New("ID MongoDB tidak valid")
    }

    mongoDoc, err := s.MongoRepo.GetByID(ctx, mongoID)
    if err != nil {
        return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
    }

    var submittedAt *time.Time
    if ref.SubmittedAt.Valid {
        submittedAt = &ref.SubmittedAt.Time
    }
    var verifiedAt *time.Time
    if ref.VerifiedAt.Valid {
        verifiedAt = &ref.VerifiedAt.Time
    }

    verifiedByStr := ""
    if ref.VerifiedBy != nil {
        verifiedByStr = ref.VerifiedBy.String()
    }

    rejectionNoteStr := ""
    if ref.RejectionNote.Valid {
        rejectionNoteStr = ref.RejectionNote.String
    }

    return &models.AchievementDetail{
        Achievement:  *mongoDoc,
        ReferenceID:  ref.ID.String(),
        Status:       ref.Status,
        SubmittedAt:  submittedAt,
        VerifiedAt:   verifiedAt,
        VerifiedBy:   verifiedByStr,
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

	if err != nil {
		return nil, fmt.Errorf("gagal mengambil referensi prestasi dari PostgreSQL: %w", err)
	}

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
	if err != nil {
		return nil, fmt.Errorf("gagal fetch detail dari MongoDB: %w", err)
	}

	if len(mongoDocs) == 0 {
		return []models.AchievementDetail{}, nil
	}

	details := make([]models.AchievementDetail, 0, len(mongoDocs))
	for _, doc := range mongoDocs {
		ref, found := refMap[doc.ID.Hex()]
		if found {
			var submittedAt *time.Time
			if ref.SubmittedAt.Valid {
				submittedAt = &ref.SubmittedAt.Time
			}
			var verifiedAt *time.Time
			if ref.VerifiedAt.Valid {
				verifiedAt = &ref.VerifiedAt.Time
			}

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
	// --- 0. Role determination and ID collection ---
	role := strings.ToLower(userRole)
	isGlobalAdmin := (role == "admin")
	var targetStudentIDs []uuid.UUID

	switch role {
	case "admin":
		targetStudentIDs = nil // Admin reads all data
	case "mahasiswa":
		// Mahasiswa (own)
		studentID, err := s.PostgreRepo.GetStudentProfileID(ctx, userID)
		if err != nil {
			return nil, errors.New("forbidden: user is not a registered student")
		}
		targetStudentIDs = []uuid.UUID{studentID}
	case "dosen wali":
		// Dosen Wali (advisee)
		lecturerProfileID, err := s.PostgreRepo.GetLecturerProfileID(ctx, userID)
		if err != nil {
			return nil, errors.New("forbidden: user is not a registered lecturer")
		}
		adviseeIDs, err := s.PostgreRepo.GetAdviseeIDs(ctx, lecturerProfileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get advisee IDs: %w", err)
		}
		targetStudentIDs = adviseeIDs
	default:
		return nil, errors.New("forbidden: role cannot access statistics")
	}

	// --- 1. Ambil Data Operasional dari PostgreSQL (Status Submissions) ---
	pqRefs, err := s.PostgreRepo.GetAllReferences(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil referensi prestasi dari postgre: %w", err)
	}

	// FILTERING PQREFS: Apply role-based filter
	filteredRefs := make([]models.AchievementReference, 0)

	// Convert targetStudentIDs to map for quick lookup if not Admin
	targetIDMap := make(map[uuid.UUID]bool)
	if !isGlobalAdmin {
		for _, id := range targetStudentIDs {
			targetIDMap[id] = true
		}
	}

	for _, ref := range pqRefs {
		// 1. Filter Deleted Status
		if ref.Status == models.StatusDeleted {
			continue
		}

		// 2. Apply Role Filter
		if isGlobalAdmin {
			filteredRefs = append(filteredRefs, ref)
		} else {
			if targetIDMap[ref.StudentID] {
				filteredRefs = append(filteredRefs, ref)
			}
		}
	}

	statsCounts := map[string]int{
		string(models.StatusDraft): 0,
		string(models.StatusSubmitted): 0,
		string(models.StatusVerified): 0,
		string(models.StatusRejected): 0,
	}

	// Hitung status pengajuan (total_submissions & status_counts) dari filteredRefs
	for _, ref := range filteredRefs {
		statusKey := string(ref.Status)
		if _, found := statsCounts[statusKey]; found {
			statsCounts[statusKey]++
		}
	}

	totalReferences := len(filteredRefs) // Total submission yang relevan dengan role

	// --- 2. Ambil Data Agregasi Prestasi dari MongoDB ---

	var statsByType []models.StatsByType
	var statsByPeriod []models.StatsByPeriod
	var topStudents []models.StatsTopStudent
	var statsByLevel []models.StatsByLevel

	if isGlobalAdmin {
		// ADMIN: Fetch global detailed stats from MongoRepo
		statsByType, err = s.MongoRepo.GetStatsByType(ctx)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil statistik by type: %w", err)
		}
		if statsByType == nil {
			statsByType = []models.StatsByType{}
		}

		statsByPeriodRaw, err := s.MongoRepo.GetStatsByYear(ctx)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil statistik by period: %w", err)
		}
		statsByPeriod = make([]models.StatsByPeriod, 0, len(statsByPeriodRaw))
		for _, raw := range statsByPeriodRaw {
			periodStr := raw.Period
			if periodStr == "" || periodStr == "0" || periodStr == "1970" {
				continue
			}
			statsByPeriod = append(statsByPeriod, models.StatsByPeriod{Period: periodStr, Count: raw.Count})
		}
		if statsByPeriod == nil {
			statsByPeriod = []models.StatsByPeriod{}
		}

		topStudents, err = s.MongoRepo.GetTopStudents(ctx)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil top students: %w", err)
		}
		if topStudents == nil {
			topStudents = []models.StatsTopStudent{}
		}

		statsByLevel, err = s.MongoRepo.GetStatsByLevel(ctx)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil statistik by level: %w", err)
		}
		if statsByLevel == nil {
			statsByLevel = []models.StatsByLevel{}
		}

	} else {
		// NON-ADMIN: Detailed aggregation is unavailable (return empty slices)
		statsByType = []models.StatsByType{}
		statsByPeriod = []models.StatsByPeriod{}
		statsByLevel = []models.StatsByLevel{}

		if role == "mahasiswa" && statsCounts[string(models.StatusVerified)] > 0 {
			studentID := targetStudentIDs[0]
			topStudents = []models.StatsTopStudent{
				{
					StudentID: studentID,
					Count:     int64(statsCounts[string(models.StatusVerified)]),
				},
			}
		} else {
			topStudents = []models.StatsTopStudent{}
		}
	}

	// Total Achievements (Hanya hitungan yang Verified dari submission)
	totalAchievements := statsCounts[string(models.StatusVerified)]

	// --- 3. Finalisasi Top Students ---
	finalTopStudents := make([]models.StatsTopStudent, 0, len(topStudents))
	for _, ts := range topStudents {
		if ts.StudentID.String() != "" && ts.StudentID.String() != "00000000-0000-0000-0000-000000000000" {
			finalTopStudents = append(finalTopStudents, ts)
		}
	}

	// --- 4. Gabungkan Hasil ke AchievementStatisticsResult ---

	result := models.AchievementStatistics{
		LastUpdated: time.Now(),

		// Data Postgre (Filtered)
		TotalSubmissions: totalReferences,
		StatusCounts: statsCounts,

		// Data Agregasi (Mongo - Global untuk Admin, Filtered/Empty untuk lainnya)
		TotalAchievements: totalAchievements,
		AchievementByType: statsByType,
		AchievementByPeriod: statsByPeriod,
		TopStudents: finalTopStudents,
		CompetitionLevelDistribution: statsByLevel,
	}

	return result, nil
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
		if ref.SubmittedAt.Valid {
			submittedAt = &ref.SubmittedAt.Time
		}

		var verifiedAt *time.Time
		if ref.VerifiedAt.Valid {
			verifiedAt = &ref.VerifiedAt.Time
		}

		var rejectionNote string
		if ref.RejectionNote.Valid {
			rejectionNote = ref.RejectionNote.String
		}

		var verifiedBy string
		if ref.VerifiedBy != nil {
			verifiedBy = ref.VerifiedBy.String()
		}

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