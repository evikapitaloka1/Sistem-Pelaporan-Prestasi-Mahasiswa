package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	
	// Pastikan path import model Anda benar (misalnya uas/app/model/mongo)
	"uas/app/model/mongo" 
)

// --- MONGODB Repository ---

// MongoAchievementRepository mendefinisikan kontrak CRUD dasar untuk MongoDB.
type MongoAchievementRepository interface {
	Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Achievement, error)
	GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error
	SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error
	DeleteByID(ctx context.Context, id primitive.ObjectID) error // Untuk Rollback
	AddAttachment(ctx context.Context, id primitive.ObjectID, attachment models.Attachment) error
}

type mongoAchievementRepo struct {
	collection *mongo.Collection
}

func NewMongoAchievementRepository(coll *mongo.Collection) MongoAchievementRepository {
	return &mongoAchievementRepo{collection: coll}
}

// Implementasi MongoDB (Tetap sama)
func (r *mongoAchievementRepo) Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error) {
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
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&achievement)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("achievement not found in MongoDB")
		}
		return nil, fmt.Errorf("mongo find failed: %w", err)
	}
	return &achievement, nil
}

func (r *mongoAchievementRepo) GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error) {
	filter := bson.M{"_id": bson.M{"$in": ids}}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo find many failed: %w", err)
	}
	defer cursor.Close(ctx)

	var achievements []models.Achievement
	if err := cursor.All(ctx, &achievements); err != nil {
		return nil, fmt.Errorf("mongo cursor decode failed: %w", err)
	}
	return achievements, nil
}

func (r *mongoAchievementRepo) UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error {
	updateData["updatedAt"] = time.Now()
	_, err := r.collection.UpdateByID(ctx, id, bson.M{"$set": updateData})
	if err != nil {
		return fmt.Errorf("mongo update failed: %w", err)
	}
	return nil
}

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

func (r *mongoAchievementRepo) DeleteByID(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *mongoAchievementRepo) AddAttachment(ctx context.Context, id primitive.ObjectID, attachment models.Attachment) error {
	update := bson.M{
		"$push": bson.M{"attachments": attachment},
		"$set": 	bson.M{"updatedAt": time.Now()},
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("mongo add attachment failed: %w", err)
	}
	return nil
}


// --- POSTGRESQL Repository ---

// PostgreAchievementRepository mendefinisikan kontrak akses data ke PostgreSQL (references & users/students).
type PostgreAchievementRepository interface {
    // Menggunakan uuid.UUID untuk semua ID
    GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) 
    GetReferenceByMongoID(ctx context.Context, mongoID string) (*models.AchievementReference, error)
    GetReferenceByID(ctx context.Context, refID uuid.UUID) (*models.AchievementReference, error) 
    CreateReference(ctx context.Context, ref *models.AchievementReference) error
    
    // ✅ PERBAIKAN 1: Menghapus parameter 'status' yang duplikat
    UpdateReferenceStatus(
        ctx context.Context, 
        refID uuid.UUID, 
        status models.AchievementStatus, // Status yang benar
        note sql.NullString, 
        verifiedBy sql.NullString,
    ) error 

    // List 
    GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error) 
    GetAllReferences(ctx context.Context) ([]models.AchievementReference, error)
    
    // Tambahan untuk Dosen Wali (FR-006)
    GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error) 
    
    // ✅ PERBAIKAN 2: Menambahkan parameter status untuk operasi Soft Delete
    UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error
}

type postgreAchievementRepo struct {
	db *sql.DB
}

func NewPostgreAchievementRepository(db *sql.DB) PostgreAchievementRepository {
	return &postgreAchievementRepo{db: db}
}

// GetStudentProfileID mengambil students.id (UUID) berdasarkan user_id.
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

// GetReferenceByMongoID mengambil referensi berdasarkan ID MongoDB.
func (r *postgreAchievementRepo) GetReferenceByMongoID(ctx context.Context, mongoID string) (*models.AchievementReference, error) {
	var ref models.AchievementReference
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
              FROM achievement_references 
              WHERE mongo_achievement_id = $1`
              
    // PENTING: Menggunakan &ref.VerifiedBy (*uuid.UUID) untuk scanning kolom UUID nullable
	err := r.db.QueryRowContext(ctx, query, mongoID).Scan(
		&ref.ID,
		&ref.StudentID,
		&ref.MongoAchievementID,
		&ref.Status,
		&ref.SubmittedAt, 
		&ref.RejectionNote,
		&ref.VerifiedBy, // <-- FIX: Scanning ke *uuid.UUID
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

// GetReferenceByID mengambil referensi berdasarkan ID PostgreSQL.
func (r *postgreAchievementRepo) GetReferenceByID(ctx context.Context, refID uuid.UUID) (*models.AchievementReference, error) {
	var ref models.AchievementReference
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
              FROM achievement_references 
              WHERE id = $1`
              
    // PENTING: Menggunakan &ref.VerifiedBy (*uuid.UUID)
	err := r.db.QueryRowContext(ctx, query, refID).Scan(
		&ref.ID,
		&ref.StudentID,
		&ref.MongoAchievementID,
		&ref.Status,
		&ref.SubmittedAt, 
		&ref.RejectionNote,
		&ref.VerifiedBy, // <-- FIX: Scanning ke *uuid.UUID
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

// CreateReference menyimpan metadata workflow ke tabel achievement_references.
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

// UpdateReferenceStatus mengupdate status workflow. (Submit, Verify, Reject)
// KODE BARU (Memperbaiki Error)
func (r *postgreAchievementRepo) UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error {
    
    // Query ini akan mengubah status menjadi StatusDeleted (yang dikirim dari service)
    query := `UPDATE achievement_references 
              SET status = $1, updated_at = NOW() 
              WHERE id = $2`
              
    // ✅ PERBAIKAN: Melewatkan 'status' sebagai argumen pertama
    res, err := r.db.ExecContext(ctx, query, status, refID) 
    
    if err != nil {
        return fmt.Errorf("postgre update status for soft delete failed: %w", err)
    }

    rowsAffected, _ := res.RowsAffected()
    if rowsAffected == 0 {
        return errors.New("cannot update reference status: id not found")
    }
    return nil
}
// GetAdviseeIDs mendapatkan semua ID Mahasiswa yang dibimbing oleh Dosen Wali ini. (FR-006)
func (r *postgreAchievementRepo) GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error) {
	var studentIDs []uuid.UUID
	query := "SELECT id FROM students WHERE advisor_id = $1"
	
	rows, err := r.db.QueryContext(ctx, query, lecturerID)
	if err != nil {
		return nil, fmt.Errorf("postgre get advisee IDs failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var studentID uuid.UUID 
		if err := rows.Scan(&studentID); err != nil {
			return nil, fmt.Errorf("postgre scan advisee ID failed: %w", err)
		}
		studentIDs = append(studentIDs, studentID)
	}
	
	return studentIDs, nil
}

// GetReferencesByStudentIDs mendapatkan semua references untuk sekumpulan Mahasiswa. (Dosen Wali View)
func (r *postgreAchievementRepo) GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error) {
	if len(studentIDs) == 0 {
		return []models.AchievementReference{}, nil
	}
	
	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references 
			  WHERE student_id = ANY($1)`
	
	rows, err := r.db.QueryContext(ctx, query, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("postgre get references by IDs failed: %w", err)
	}
	defer rows.Close()
	
	var refs []models.AchievementReference
	for rows.Next() {
		var ref models.AchievementReference
        // PENTING: Menggunakan &ref.VerifiedBy (*uuid.UUID)
		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&ref.VerifiedBy, // <-- FIX: Scanning ke *uuid.UUID
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}
		refs = append(refs, ref)
	}
	
	return refs, nil
}

// GetAllReferences mendapatkan semua references. (Admin View)
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
        // PENTING: Menggunakan &ref.VerifiedBy (*uuid.UUID)
		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&ref.VerifiedBy, // <-- FIX: Scanning ke *uuid.UUID
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}
		refs = append(refs, ref)
	}
	
	return refs, nil
}

// UpdateReferenceForDelete digunakan saat Mahasiswa menghapus prestasi draft (FR-005)
// Add this function to your postgreAchievementRepo struct implementation block
func (r *postgreAchievementRepo) UpdateReferenceStatus(
	ctx context.Context, 
	refID uuid.UUID, 
	status models.AchievementStatus, 
	note sql.NullString, 
	verifiedBy sql.NullString,
) error {
	query := `
		UPDATE achievement_references
		SET status = $1, rejection_note = $2, verified_by = $3, updated_at = NOW(),
			submitted_at = CASE WHEN $1 = 'submitted' THEN NOW() ELSE submitted_at END,
			verified_at = CASE WHEN $1 = 'verified' THEN NOW() ELSE verified_at END
		WHERE id = $4
	`
	// Note: verifiedBy (sql.NullString) here must hold a valid UUID string or NULL.
	_, err := r.db.ExecContext(ctx, query, status, note, verifiedBy, refID)
	if err != nil {
		return fmt.Errorf("postgre update status failed: %w", err)
	}
	return nil
}