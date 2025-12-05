package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"github.com/lib/pq"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	// Pastikan path import model Anda benar
	models "uas/app/model/mongo" 
)

// --- MONGODB Repository ---

// MongoAchievementRepository mendefinisikan kontrak CRUD dasar untuk MongoDB.
type MongoAchievementRepository interface {
	Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Achievement, error)
	GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error
	
	// ✅ SOFT DELETE
	SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error
	
	// Hard Delete (Tetap dipertahankan, namun Soft Delete yang digunakan)
	DeleteByID(ctx context.Context, id primitive.ObjectID) error 
	
	// Attachment
	AddAttachment(ctx context.Context, achievementID primitive.ObjectID, attachment *models.Attachment) error
	
	GetStatsByType(ctx context.Context) ([]models.StatsByType, error)
    GetStatsByYear(ctx context.Context) ([]models.StatsByPeriod, error)
    GetStatsByLevel(ctx context.Context) ([]models.StatsByLevel, error)
    GetTopStudents(ctx context.Context) ([]models.StatsTopStudent, error)
}

type mongoAchievementRepo struct {
	collection *mongo.Collection
}

func NewMongoAchievementRepository(coll *mongo.Collection) MongoAchievementRepository {
	// Baris ini sekarang lolos kompilasi karena DeleteByID sudah dihapus dari interface
	return &mongoAchievementRepo{collection: coll}
}

// Implementasi MONGODB
// Di file mongo_achievement_repo.go (atau sejenisnya)

// GetStatsByType mengambil total prestasi per tipe dari MongoDB
func (r *mongoAchievementRepo) GetStatsByType(ctx context.Context) ([]models.StatsByType, error) {
	pipeline := []bson.D{
		// 1. Grouping berdasarkan achievementType
		{{"$group", bson.D{
			{"_id", "$achievementType"}, // Field _id akan di-map ke models.StatsByType.Type
			{"count", bson.D{{"$sum", 1}}},
		}}},
		// 2. Sort by count (Optional: agar yang terbanyak di atas)
		{{"$sort", bson.D{{"count", -1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mongo aggregate stats by type failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.StatsByType
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo decode stats by type failed: %w", err)
	}

	return results, nil
}

// --- 2. Total prestasi per periode (Tahun EventDate) ---
func (r *mongoAchievementRepo) GetStatsByYear(ctx context.Context) ([]models.StatsByPeriod, error) {
	pipeline := []bson.D{
		// 1. AddFields: Ekstrak tahun dari EventDate
		{{"$addFields", bson.D{
			{"eventYear", bson.D{{"$year", "$details.eventDate"}}},
		}}},
		// 2. Grouping berdasarkan tahun yang diekstrak
		{{"$group", bson.D{
			{"_id", "$eventYear"}, // Field _id akan di-map ke models.StatsByYear.Year
			{"count", bson.D{{"$sum", 1}}},
		}}},
		// 3. Sort by year (Optional: dari tahun terbaru)
		{{"$sort", bson.D{{"_id", -1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mongo aggregate stats by year failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.StatsByPeriod
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo decode stats by year failed: %w", err)
	}

	return results, nil
}

// --- 3. Distribusi tingkat kompetisi (CompetitionLevel) ---
func (r *mongoAchievementRepo) GetStatsByLevel(ctx context.Context) ([]models.StatsByLevel, error) {
	// Catatan: Hanya prestasi yang memiliki details.competitionLevel yang akan terhitung
	pipeline := []bson.D{
		// 1. Match: Hanya dokumen yang memiliki competitionLevel (untuk menghilangkan non-kompetisi)
		{{"$match", bson.D{{"details.competitionLevel", bson.D{{"$exists", true}, {"$ne", nil}}}}}}, 
		// 2. Grouping berdasarkan competitionLevel
		{{"$group", bson.D{
			{"_id", "$details.competitionLevel"}, // Field _id akan di-map ke models.StatsByLevel.Level
			{"count", bson.D{{"$sum", 1}}},
		}}},
		// 3. Sort by count
		{{"$sort", bson.D{{"count", -1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mongo aggregate stats by level failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.StatsByLevel
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo decode stats by level failed: %w", err)
	}

	return results, nil
}

// --- 4. Top mahasiswa berprestasi (Top Students) ---
func (r *mongoAchievementRepo) GetTopStudents(ctx context.Context) ([]models.StatsTopStudent, error) {
	// Catatan: Agar output ini lengkap (NIM/Nama), Anda perlu me-JOIN dengan data mahasiswa
	// (misalnya dari PostgreSQL atau service eksternal) setelah agregasi ini.
	// Di sini kita hanya mengembalikan StudentID (UUID) dan Count.

	pipeline := []bson.D{
		// 1. Grouping berdasarkan StudentID
		{{"$group", bson.D{
			{"_id", "$studentId"}, // Field _id akan di-map ke models.StatsTopStudents.StudentID
			{"count", bson.D{{"$sum", 1}}},
		}}},
		// 2. Sort by count (limit 10 besar)
		{{"$sort", bson.D{{"count", -1}}}},
		// 3. Limit (Ambil 10 teratas)
		{{"$limit", 10}}, 
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("mongo aggregate top students failed: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.StatsTopStudent
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("mongo decode top students failed: %w", err)
	}

	// WARNING: StudentNIM dan StudentName di struct models.StatsTopStudents tidak akan terisi 
	// karena data tersebut TIDAK ada di collection prestasi MongoDB. 
	// Anda harus melengkapi field tersebut di Service Layer (menggunakan StudentID) dengan 
	// memanggil service data Mahasiswa.

	return results, nil
}
func (r *mongoAchievementRepo) Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error) {
    if achievement.ID.IsZero() {
        achievement.ID = primitive.NewObjectID()
    }
    if achievement.CreatedAt.IsZero() {
        achievement.CreatedAt = time.Now()
    }
    if achievement.UpdatedAt.IsZero() {
        achievement.UpdatedAt = time.Now()
    }

    result, err := r.collection.InsertOne(ctx, achievement)
    if err != nil {
        return nil, fmt.Errorf("mongo insert failed: %w", err)
    }

    id := result.InsertedID.(primitive.ObjectID)
    return &id, nil
}


func (r *mongoAchievementRepo) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Achievement, error) {
	var achievement models.Achievement
	
	// Filter untuk soft delete dan mencoba mengabaikan data lama yang rusak (string kosong)
	filter := bson.M{"_id": id, "deletedAt": nil} 
	
	err := r.collection.FindOne(ctx, filter).Decode(&achievement)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("achievement not found or already deleted in MongoDB")
		}
		// 🎯 PERBAIKAN: Jika error decoding terjadi (termasuk masalah parsing time), catat dengan jelas.
		// Penggunaan NullableTime di model seharusnya mengatasi ini, tapi kita pastikan error tetap di-wrap.
		return nil, fmt.Errorf("mongo find failed: error decoding document for ID %s: %w", id.Hex(), err)
	}
	return &achievement, nil
}

func (r *mongoAchievementRepo) GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error) {
	// Filter untuk soft delete
	filter := bson.M{"_id": bson.M{"$in": ids}, "deletedAt": nil} 
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo find many failed: %w", err)
	}
	defer cursor.Close(ctx)

	var achievements []models.Achievement
	// 🎯 PERBAIKAN: Error ini adalah error Time Decoding. Fix ada di Model (NullableTime).
	// Di sini, kita hanya memastikan kita menangkap error tersebut dengan pesan yang lebih informatif.
	if err := cursor.All(ctx, &achievements); err != nil {
		// Jika NullableTime gagal, error akan muncul di sini.
		return nil, fmt.Errorf("mongo cursor decode failed while fetching multiple documents: %w", err)
	}
	return achievements, nil
}


func (r *mongoAchievementRepo) UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error {
	// Pastikan field deletedAt tidak diubah
	delete(updateData, "deletedAt")
	updateData["updatedAt"] = time.Now()
	_, err := r.collection.UpdateByID(ctx, id, bson.M{"$set": updateData})
	if err != nil {
		return fmt.Errorf("mongo update failed: %w", err)
	}
	return nil
}

// ✅ IMPLEMENTASI SOFT DELETE
func (r *mongoAchievementRepo) SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error {
	update := bson.M{"$set": bson.M{
		"deletedAt": time.Now(),
		"updatedAt": time.Now(),
	}}
	
	_, err := r.collection.UpdateByID(ctx, id, update) 
	
	if err != nil {
		return fmt.Errorf("mongo soft delete failed: %w", err)
	}
	return nil
}

// IMPLEMENTASI HARD DELETE
func (r *mongoAchievementRepo) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
    filter := bson.M{"_id": id}
    _, err := r.collection.DeleteOne(ctx, filter)
    if err != nil {
        return fmt.Errorf("mongo hard delete failed: %w", err)
    }
    return nil
}

// IMPLEMENTASI ADD ATTACHMENT
func (r *mongoAchievementRepo) AddAttachment(ctx context.Context, id primitive.ObjectID, attachment *models.Attachment) error {
    
    filter := bson.M{"_id": id}

    // [Langkah 1: Inisialisasi array jika null]
    initFilter := bson.M{"_id": id, "attachments": nil}
    initUpdate := bson.M{"$set": bson.M{"attachments": []models.Attachment{}},} 
    _, err := r.collection.UpdateOne(ctx, initFilter, initUpdate)
    if err != nil {
        return fmt.Errorf("mongo failed during array initialization check: %w", err)
    }

    // [Langkah 2: Push item baru]
	pushUpdate := bson.M{
		"$push": bson.M{
			"attachments": attachment, // Masih berfungsi karena operator $push menerima pointer
		},
		"$set": bson.M{
			"updatedAt": time.Now(), 
		},
	}
    
	result, err := r.collection.UpdateOne(ctx, filter, pushUpdate)
	
	if err != nil {
		return fmt.Errorf("mongo failed to push attachment: %w", err)
	}
    
    if result.MatchedCount == 0 {
        return errors.New("achievement document not found during push operation")
    }

	return nil
}
// --- POSTGRESQL Repository ---

// PostgreAchievementRepository mendefinisikan kontrak akses data ke PostgreSQL.
type PostgreAchievementRepository interface {
	GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) 
	GetReferenceByMongoID(ctx context.Context, mongoID string) (*models.AchievementReference, error)
	GetReferenceByID(ctx context.Context, refID uuid.UUID) (*models.AchievementReference, error)
	CreateReference(ctx context.Context, ref *models.AchievementReference) error
	UpdateReferenceStatus(ctx context.Context, refID uuid.UUID, status models.AchievementStatus, note sql.NullString, verifiedBy sql.NullString) error

	// List 
	GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error) 
	GetAllReferences(ctx context.Context) ([]models.AchievementReference, error)
	
	// Tambahan untuk Dosen Wali (FR-006)
	GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error) 
	GetLecturerProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) 
	UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error
	GetHistoryByMongoID(ctx context.Context, mongoID string) ([]models.AchievementReference, error)
	

}
	// ✅ SOFT DELETE: Status Update
	

type postgreAchievementRepo struct {
	db *sql.DB
}

func NewPostgreAchievementRepository(db *sql.DB) PostgreAchievementRepository {
	return &postgreAchievementRepo{db: db}
}

// Implementasi POSTGRESQL

func (r *postgreAchievementRepo) GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var studentID uuid.UUID
	query := "SELECT id FROM students WHERE user_id = $1"
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&studentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, errors.New("profil mahasiswa tidak ditemukan")
		}
		return uuid.Nil, fmt.Errorf("postgre query failed: %w", err)
	}
	return studentID, nil
}

func (r *postgreAchievementRepo) GetReferenceByMongoID(ctx context.Context, mongoID string) (*models.AchievementReference, error) {
	var ref models.AchievementReference
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references 
			  WHERE mongo_achievement_id = $1`
	err := r.db.QueryRowContext(ctx, query, mongoID).Scan(
		&ref.ID,
		&ref.StudentID,
		&ref.MongoAchievementID,
		&ref.Status,
		&ref.SubmittedAt,
		&ref.RejectionNote,
		&ref.VerifiedBy,
		&ref.VerifiedAt,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("achievement reference not found")
		}
		return nil, fmt.Errorf("postgre query failed: %w", err)
	}
	return &ref, nil
}

func (r *postgreAchievementRepo) GetReferenceByID(ctx context.Context, refID uuid.UUID) (*models.AchievementReference, error) {
	var ref models.AchievementReference
	// 🛑 Variabel filter dihapus dari sini

	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references 
			  WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, refID).Scan(
		&ref.ID,
		&ref.StudentID,
		&ref.MongoAchievementID,
		&ref.Status,
		&ref.SubmittedAt,
		&ref.RejectionNote,
		&ref.VerifiedBy,
		&ref.VerifiedAt,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("achievement reference not found")
		}
		return nil, fmt.Errorf("postgre query failed: %w", err)
	}
	return &ref, nil
}

func (r *postgreAchievementRepo) CreateReference(ctx context.Context, ref *models.AchievementReference) error {
	query := `
		INSERT INTO achievement_references (id, student_id, mongo_achievement_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		ref.ID,
		ref.StudentID,
		ref.MongoAchievementID,
		ref.Status,
		ref.CreatedAt,
		ref.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postgre insert reference failed: %w", err)
	}
	return nil
}

// uas/app/repository/mongo/postgreAchievementRepo.go (Contoh Implementasi)

// Asumsikan ini adalah bagian dari struct PostgreAchievementRepositoryImpl
func (r *postgreAchievementRepo) UpdateReferenceStatus(
	ctx context.Context, 
	refID uuid.UUID, 
	status models.AchievementStatus, 
	rejectionNote sql.NullString, 
	verifiedBy sql.NullString,
) error {
    
    // Pastikan 'verified_by' di database adalah tipe UUID
	query := `
		UPDATE achievement_references 
		SET 
			status = $1, 
			rejection_note = $2, 
			verified_by = $3, -- Kolom ini
			updated_at = $4,
			submitted_at = CASE WHEN $1 = 'submitted' THEN NOW() ELSE submitted_at END,
			verified_at = CASE WHEN $1 = 'verified' OR $1 = 'rejected' THEN NOW() ELSE verified_at END
		WHERE id = $5
	`
    
    // Jika statusnya bukan verified/rejected, di service layer, 
    // verifiedBy akan dikirim sebagai Valid: false, sehingga Repository mengirimkan NULL.
    // Jika statusnya verified/rejected, di service layer, verifiedBy adalah ID Dosen valid dengan Valid: true.
    
	_, err := r.db.ExecContext(
		ctx, 
		query, 
		status, 
		rejectionNote, 
		verifiedBy, // Menggunakan sql.NullString, yang akan menjadi NULL jika Valid=false atau ID UUID String jika Valid=true
		time.Now(), 
		refID,
	)
    
	if err != nil {
		return fmt.Errorf("failed to update reference status to '%s': %w", status, err)
	}

	return nil
}

func (r *postgreAchievementRepo) GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error) {
    var studentIDs []uuid.UUID
    
    // 💡 PERBAIKAN: Mengganti tabel yang tidak ada ('student_profiles') 
    // dengan tabel yang diasumsikan ada dan memiliki relasi ('students').
    // Mengambil kolom 'id' (yang merupakan ID Mahasiswa)
    query := "SELECT id FROM students WHERE advisor_id = $1" 
    
    rows, err := r.db.QueryContext(ctx, query, lecturerID)
    if err != nil {
        // Jika error (misalnya tabel 'students' juga tidak ada, atau koneksi), 
        // error akan di-wrap di sini.
        return nil, fmt.Errorf("postgre get advisee IDs failed (using students table): %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var studentID uuid.UUID
        if err := rows.Scan(&studentID); err != nil {
            return nil, fmt.Errorf("postgre scan advisee ID failed: %w", err)
        }
        studentIDs = append(studentIDs, studentID)
    }
    
    // Pastikan data yang diambil dari DB sudah ada
    if len(studentIDs) == 0 {
        // Return nil error, tapi list kosong jika memang tidak ada bimbingan
        return studentIDs, nil 
    }
    
    return studentIDs, nil
}

func (r *postgreAchievementRepo) GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error) {

	if len(studentIDs) == 0 {
		return []models.AchievementReference{}, nil
	}

	query := `
		SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at
		FROM achievement_references
		WHERE student_id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(studentIDs))
	if err != nil {
		return nil, fmt.Errorf("postgre get references by IDs failed: %w", err)
	}
	defer rows.Close()

	var refs []models.AchievementReference

	for rows.Next() {
		var ref models.AchievementReference
		var verifiedByStr sql.NullString // UUID nullable sebagai string (Intermediate)

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr, // <---- PERBAIKAN: Scan ke intermediate
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}

		// Convert verifiedBy string ke *uuid.UUID
		var verifiedByID *uuid.UUID
		if verifiedByStr.Valid {
			uid, err := uuid.Parse(verifiedByStr.String)
			if err == nil {
				verifiedByID = &uid
			}
		}
		ref.VerifiedBy = verifiedByID // Assigment ke pointer UUID

		refs = append(refs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return refs, nil
}
func (r *postgreAchievementRepo) GetAllReferences(ctx context.Context) ([]models.AchievementReference, error) {
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgre get all references failed: %w", err)
	}
	defer rows.Close()

	var refs []models.AchievementReference
	for rows.Next() {
		var ref models.AchievementReference
		var verifiedByStr sql.NullString // <---- PERBAIKAN: Intermediate string

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr, // <---- PERBAIKAN: Scan ke intermediate
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}
        
        // Konversi ke *uuid.UUID
        var verifiedByID *uuid.UUID
        if verifiedByStr.Valid {
            uid, _ := uuid.Parse(verifiedByStr.String) // Abaikan error parse UUID (seharusnya sudah dijamin valid)
            verifiedByID = &uid
        }
        ref.VerifiedBy = verifiedByID // Assign

		refs = append(refs, ref)
	}

	return refs, nil
}
// ✅ IMPLEMENTASI SOFT DELETE POSTGRES
func (r *postgreAchievementRepo) UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error {

	// Siapkan NullString untuk rejection_note ($2) dan verified_by ($3)
	// Saat soft delete, kedua field ini disarankan NULL
	nullString := sql.NullString{Valid: false}
	
	// Query ini akan mengubah status menjadi StatusDeleted
	query := `
	UPDATE achievement_references
	SET status = $1::achievement_status, -- $1
		rejection_note = $2,            -- $2 = nullString
		verified_by = $3,               -- $3 = nullString
		updated_at = NOW(),
		submitted_at = CASE WHEN $1::text = 'submitted' THEN NOW() ELSE submitted_at END, 
		verified_at = CASE WHEN $1::text = 'verified' THEN NOW() ELSE verified_at END
	WHERE id = $4
` 	
	// 🛑 PERBAIKAN: Kirim 4 parameter (status, nullString, nullString, refID)
	res, err := r.db.ExecContext(ctx, query, status, nullString, nullString, refID)
	if err != nil {
		return fmt.Errorf("postgre update status for soft delete failed: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("cannot update reference status: id not found")
	}
	return nil
}
// Di dalam struct postgreAchievementRepo

// Di implementasi GetLecturerProfileID (Postgre Repository)

func (r *postgreAchievementRepo) GetHistoryByMongoID(ctx context.Context, mongoID string) ([]models.AchievementReference, error) {
	var refs []models.AchievementReference
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references 
			  WHERE mongo_achievement_id = $1`

	rows, err := r.db.QueryContext(ctx, query, mongoID)
	if err != nil {
		return nil, fmt.Errorf("postgre query GetHistoryByMongoID failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ref models.AchievementReference
		var verifiedByStr sql.NullString // <---- PERBAIKAN: Intermediate string

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr, // <---- PERBAIKAN: Scan ke intermediate
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan history failed: %w", err)
		}
        
        // Konversi ke *uuid.UUID
        var verifiedByID *uuid.UUID
        if verifiedByStr.Valid {
            uid, _ := uuid.Parse(verifiedByStr.String)
            verifiedByID = &uid
        }
        ref.VerifiedBy = verifiedByID // Assign
		
		refs = append(refs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during history iteration: %w", err)
	}

	return refs, nil
}

func (r *postgreAchievementRepo) GetLecturerProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var lecturerProfileID uuid.UUID

	// Query untuk mendapatkan ID dosen dari tabel 'lecturers' berdasarkan 'user_id'
	query := `SELECT id FROM lecturers WHERE user_id = $1`

	// Eksekusi query dan coba scan hasilnya ke lecturerProfileID
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&lecturerProfileID)

	if err != nil {
		// 1. Cek jika dosen tidak ditemukan (sql.ErrNoRows)
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, errors.New("profil dosen tidak ditemukan untuk user ini")
		}
		// 2. Error database lainnya
		return uuid.Nil, fmt.Errorf("postgre query GetLecturerProfileID failed: %w", err)
	}

	// Jika berhasil, kembalikan ID profil dosen (lecturers.id)
	return lecturerProfileID, nil
}