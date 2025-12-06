package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	models "uas/app/model/postgres"
)

type StudentRepository interface {
	GetByID(ctx context.Context, studentID uuid.UUID) (*models.Student, error)
	GetAll(ctx context.Context) ([]models.Student, error)
	Create(ctx context.Context, student *models.Student) error
	UpdateAdvisor(ctx context.Context, studentID uuid.UUID, newAdvisorID uuid.UUID) error
	FindStudentByUserID(ctx context.Context, userID uuid.UUID) (*models.Student, error)
}

type studentRepo struct {
	db *sql.DB
}

func NewStudentRepository(db *sql.DB) StudentRepository {
	return &studentRepo{db: db}
}

func (r *studentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Student, error) {
	var s models.Student
	query := `SELECT id, user_id, student_id, program_study, academic_year, advisor_id, created_at 
			  FROM students WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.UserID, &s.StudentID, &s.ProgramStudy, &s.AcademicYear, &s.AdvisorID, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("student not found")
		}
		return nil, fmt.Errorf("failed to get student by ID: %w", err)
	}
	return &s, nil
}

func (r *studentRepo) GetAll(ctx context.Context) ([]models.Student, error) {
	query := `SELECT id, user_id, student_id, program_study, academic_year, advisor_id, created_at FROM students`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all students: %w", err)
	}
	defer rows.Close()

	var students []models.Student
	for rows.Next() {
		var s models.Student
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.StudentID, &s.ProgramStudy, &s.AcademicYear, &s.AdvisorID, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan student: %w", err)
		}
		students = append(students, s)
	}
	return students, nil
}

func (r *studentRepo) Create(ctx context.Context, student *models.Student) error {
	query := `INSERT INTO students (id, user_id, student_id, program_study, academic_year, advisor_id, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		student.ID, student.UserID, student.StudentID, student.ProgramStudy, student.AcademicYear, student.AdvisorID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create student: %w", err)
	}
	return nil
}

func (r *studentRepo) UpdateAdvisor(ctx context.Context, studentID uuid.UUID, newAdvisorID uuid.UUID) error {
	query := `UPDATE students SET advisor_id = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, newAdvisorID, studentID)
	if err != nil {
		return fmt.Errorf("failed to update student advisor: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return errors.New("student ID not found")
	}
	return nil
}

func (r *studentRepo) FindStudentByUserID(ctx context.Context, userID uuid.UUID) (*models.Student, error) {

	query := `
		SELECT 
			id, 
			user_id, 
			student_id, 
			program_study, 
			academic_year, 
			advisor_id, 
			created_at
		FROM 
			students
		WHERE 
			user_id = $1
		LIMIT 1;
	`

	var s models.Student

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&s.ID,
		&s.UserID,
		&s.StudentID,
		&s.ProgramStudy,
		&s.AcademicYear,
		&s.AdvisorID,
		&s.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("student not found for the given user ID")
		}
		return nil, err
	}

	return &s, nil
}