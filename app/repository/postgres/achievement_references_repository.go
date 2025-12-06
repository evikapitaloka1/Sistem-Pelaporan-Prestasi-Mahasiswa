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

	"uas/app/model/mongo"
)

// --- MONGODB Repository ---

type MongoAchievementRepository interface {
	Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Achievement, error)
	GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error
	SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error
	DeleteByID(ctx context.Context, id primitive.ObjectID) error
	AddAttachment(ctx context.Context, id primitive.ObjectID, attachment *models.Attachment) error
}

type mongoAchievementRepo struct {
	collection *mongo.Collection
}

func NewMongoAchievementRepository(coll *mongo.Collection) MongoAchievementRepository {
	return &mongoAchievementRepo{collection: coll}
}

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
	filter := bson.M{"_id": bson.M{"$in": ids}, "deletedAt": nil}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("mongo find many failed: %w", err)
	}
	defer cursor.Close(ctx)

	var achievements []models.Achievement
	if err := cursor.All(ctx, &achievements); err != nil {
		return nil, fmt.Errorf("mongo cursor decode failed while fetching multiple documents: %w", err)
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

func (r *mongoAchievementRepo) AddAttachment(ctx context.Context, id primitive.ObjectID, attachment *models.Attachment) error {

	filter := bson.M{"_id": id}

	initFilter := bson.M{"_id": id, "attachments": nil}
	initUpdate := bson.M{"$set": bson.M{"attachments": []models.Attachment{}}}
	_, err := r.collection.UpdateOne(ctx, initFilter, initUpdate)
	if err != nil {
		return fmt.Errorf("mongo failed during array initialization check: %w", err)
	}

	pushUpdate := bson.M{
		"$push": bson.M{
			"attachments": attachment,
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

type PostgreAchievementRepository interface {
	GetStudentProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetReferenceByMongoID(ctx context.Context, mongoID string) (*models.AchievementReference, error)
	GetReferenceByID(ctx context.Context, refID uuid.UUID) (*models.AchievementReference, error)
	CreateReference(ctx context.Context, ref *models.AchievementReference) error
	GetHistoryByMongoID(ctx context.Context, mongoID string) ([]models.AchievementReference, error)
	UpdateReferenceStatus(
		ctx context.Context,
		refID uuid.UUID,
		status models.AchievementStatus,
		note sql.NullString,
		verifiedBy sql.NullString,
	) error
	GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error)
	GetAllReferences(ctx context.Context) ([]models.AchievementReference, error)
	GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error)
	UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error
	GetLecturerProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type postgreAchievementRepo struct {
	db *sql.DB
}

func NewPostgreAchievementRepository(db *sql.DB) PostgreAchievementRepository {
	return &postgreAchievementRepo{db: db}
}

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

func (r *postgreAchievementRepo) UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error {

	query := `
	UPDATE achievement_references
	SET status = $1::achievement_status,
		rejection_note = NULL,
		verified_by = NULL,
		verified_at = NULL,
		submitted_at = submitted_at,
		updated_at = NOW()
	WHERE id = $2
	`

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

func (r *postgreAchievementRepo) GetReferencesByStudentIDs(
	ctx context.Context,
	studentIDs []uuid.UUID,
) ([]models.AchievementReference, error) {

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
		var verifiedBy sql.NullString

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedBy,
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}

		if verifiedBy.Valid {
			uid, err := uuid.Parse(verifiedBy.String)
			if err == nil {
				ref.VerifiedBy = &uid
			} else {
				ref.VerifiedBy = nil
			}
		} else {
			ref.VerifiedBy = nil
		}

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
		if err := rows.Scan(
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
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

func (r *postgreAchievementRepo) UpdateReferenceStatus(
	ctx context.Context,
	refID uuid.UUID,
	status models.AchievementStatus,
	note sql.NullString,
	verifiedBy sql.NullString,
) error {

	query := `
		UPDATE achievement_references
		SET 
			status = $1::achievement_status,
			rejection_note = $2,
			verified_by = $3,
			updated_at = NOW(),

			submitted_at = CASE 
				WHEN $1::text = 'submitted' THEN NOW()
				ELSE submitted_at 
			END,

			verified_at = CASE
				WHEN $1::text IN ('verified', 'rejected') THEN NOW()
				ELSE verified_at
			END
		WHERE id = $4;
	`

	res, err := r.db.ExecContext(ctx, query,
		string(status),
		note,
		verifiedBy,
		refID,
	)
	if err != nil {
		return fmt.Errorf("failed to update achievement status: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New("achievement reference ID not found")
	}

	return nil
}

func (r *postgreAchievementRepo) GetLecturerProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var lecturerProfileID uuid.UUID

	query := `SELECT id FROM lecturers WHERE user_id = $1`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(&lecturerProfileID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, errors.New("profil dosen tidak ditemukan untuk user ini")
		}
		return uuid.Nil, fmt.Errorf("postgre query GetLecturerProfileID failed: %w", err)
	}

	return lecturerProfileID, nil
}

func (r *postgreAchievementRepo) GetHistoryByMongoID(ctx context.Context, mongoID string) ([]models.AchievementReference, error) {
	var refs []models.AchievementReference

	query := `SELECT id, student_id, mongo_achievement_id, status, submitted_at, rejection_note, verified_by, verified_at, created_at, updated_at 
			  FROM achievement_references 
			  WHERE mongo_achievement_id = $1`

	rows, err := r.db.QueryContext(ctx, query, mongoID)
	if err != nil {
		return nil, fmt.Errorf("postgre get history failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ref models.AchievementReference
		if err := rows.Scan(
			&ref.ID, &ref.StudentID, &ref.MongoAchievementID, &ref.Status, &ref.SubmittedAt,
			&ref.RejectionNote, &ref.VerifiedBy, &ref.VerifiedAt, &ref.CreatedAt, &ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan history failed: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, nil
}