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

	models "uas/app/model/mongo"
)

// --- MONGODB Repository ---

type MongoAchievementRepository interface {
	Create(ctx context.Context, achievement *models.Achievement) (*primitive.ObjectID, error)
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Achievement, error)
	GetByIDs(ctx context.Context, ids []primitive.ObjectID) ([]models.Achievement, error)
	UpdateByID(ctx context.Context, id primitive.ObjectID, updateData bson.M) error
	SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error
	DeleteByID(ctx context.Context, id primitive.ObjectID) error
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
	return &mongoAchievementRepo{collection: coll}
}

// Implementasi MONGODB

func (r *mongoAchievementRepo) GetStatsByType(ctx context.Context) ([]models.StatsByType, error) {
	pipeline := []bson.D{
		{{"$group", bson.D{
			{"_id", "$achievementType"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
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

func (r *mongoAchievementRepo) GetStatsByYear(ctx context.Context) ([]models.StatsByPeriod, error) {
	pipeline := []bson.D{
		{{"$addFields", bson.D{
			{"eventYear", bson.D{{"$year", "$details.eventDate"}}},
		}}},
		{{"$group", bson.D{
			{"_id", "$eventYear"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
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

func (r *mongoAchievementRepo) GetStatsByLevel(ctx context.Context) ([]models.StatsByLevel, error) {
	pipeline := []bson.D{
		{{"$match", bson.D{{"details.competitionLevel", bson.D{{"$exists", true}, {"$ne", nil}}}}}},
		{{"$group", bson.D{
			{"_id", "$details.competitionLevel"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
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

func (r *mongoAchievementRepo) GetTopStudents(ctx context.Context) ([]models.StatsTopStudent, error) {
	pipeline := []bson.D{
		{{"$group", bson.D{
			{"_id", "$studentId"},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"count", -1}}}},
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

	filter := bson.M{"_id": id, "deletedAt": nil}

	err := r.collection.FindOne(ctx, filter).Decode(&achievement)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("achievement not found or already deleted in MongoDB")
		}
		return nil, fmt.Errorf("mongo find failed: error decoding document for ID %s: %w", id.Hex(), err)
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
	delete(updateData, "deletedAt")
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
	filter := bson.M{"_id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("mongo hard delete failed: %w", err)
	}
	return nil
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
	UpdateReferenceStatus(ctx context.Context, refID uuid.UUID, status models.AchievementStatus, note sql.NullString, verifiedBy sql.NullString) error
	GetReferencesByStudentIDs(ctx context.Context, studentIDs []uuid.UUID) ([]models.AchievementReference, error)
	GetAllReferences(ctx context.Context) ([]models.AchievementReference, error)
	GetAdviseeIDs(ctx context.Context, lecturerID uuid.UUID) ([]uuid.UUID, error)
	GetLecturerProfileID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error
	GetHistoryByMongoID(ctx context.Context, mongoID string) ([]models.AchievementReference, error)
}

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

func (r *postgreAchievementRepo) UpdateReferenceStatus(
	ctx context.Context,
	refID uuid.UUID,
	status models.AchievementStatus,
	rejectionNote sql.NullString,
	verifiedBy sql.NullString,
) error {
	query := `
		UPDATE achievement_references 
		SET 
			status = $1, 
			rejection_note = $2, 
			verified_by = $3,
			updated_at = $4,
			submitted_at = CASE WHEN $1 = 'submitted' THEN NOW() ELSE submitted_at END,
			verified_at = CASE WHEN $1 = 'verified' OR $1 = 'rejected' THEN NOW() ELSE verified_at END
		WHERE id = $5
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		status,
		rejectionNote,
		verifiedBy,
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

	query := "SELECT id FROM students WHERE advisor_id = $1"

	rows, err := r.db.QueryContext(ctx, query, lecturerID)
	if err != nil {
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

	if len(studentIDs) == 0 {
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
		var verifiedByStr sql.NullString

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr,
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}

		var verifiedByID *uuid.UUID
		if verifiedByStr.Valid {
			uid, err := uuid.Parse(verifiedByStr.String)
			if err == nil {
				verifiedByID = &uid
			}
		}
		ref.VerifiedBy = verifiedByID

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
		var verifiedByStr sql.NullString

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr,
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan reference failed: %w", err)
		}

		var verifiedByID *uuid.UUID
		if verifiedByStr.Valid {
			uid, _ := uuid.Parse(verifiedByStr.String)
			verifiedByID = &uid
		}
		ref.VerifiedBy = verifiedByID

		refs = append(refs, ref)
	}

	return refs, nil
}

func (r *postgreAchievementRepo) UpdateReferenceForDelete(ctx context.Context, refID uuid.UUID, status models.AchievementStatus) error {

	nullString := sql.NullString{Valid: false}

	query := `
	UPDATE achievement_references
	SET status = $1::achievement_status,
		rejection_note = $2,
		verified_by = $3,
		updated_at = NOW(),
		submitted_at = CASE WHEN $1::text = 'submitted' THEN NOW() ELSE submitted_at END, 
		verified_at = CASE WHEN $1::text = 'verified' THEN NOW() ELSE verified_at END
	WHERE id = $4
`
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
		var verifiedByStr sql.NullString

		if err := rows.Scan(
			&ref.ID,
			&ref.StudentID,
			&ref.MongoAchievementID,
			&ref.Status,
			&ref.SubmittedAt,
			&ref.RejectionNote,
			&verifiedByStr,
			&ref.VerifiedAt,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("postgre scan history failed: %w", err)
		}

		var verifiedByID *uuid.UUID
		if verifiedByStr.Valid {
			uid, _ := uuid.Parse(verifiedByStr.String)
			verifiedByID = &uid
		}
		ref.VerifiedBy = verifiedByID

		refs = append(refs, ref)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during history iteration: %w", err)
	}

	return refs, nil
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