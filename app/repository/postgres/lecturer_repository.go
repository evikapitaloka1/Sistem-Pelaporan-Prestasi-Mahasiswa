package repository

import (
	"context"
	"database/sql"
	"errors"
	
	"github.com/google/uuid"
	"uas/app/model/postgres" // Pastikan path ini sesuai dengan struktur project Anda
)

// LecturerRepository mendefinisikan operasi data untuk Dosen.
type LecturerRepository interface {
	// ListAllLecturers mengambil semua data Dosen.
	ListAllLecturers(ctx context.Context) ([]model.Lecturer, error)
	
	// FindLecturerByID mencari Dosen berdasarkan ID.
	FindLecturerByID(ctx context.Context, id uuid.UUID) (*model.Lecturer, error)
	
	// ListAdviseesByLecturerID mengambil daftar mahasiswa bimbingan dari ID Dosen.
	ListAdviseesByLecturerID(ctx context.Context, lecturerID uuid.UUID) ([]model.Advisee, error)
	FindLecturerByUserID(ctx context.Context, userID uuid.UUID) (*model.Lecturer, error)
}

type lecturerRepo struct {
	db *sql.DB
}

// NewLecturerRepository membuat instance baru LecturerRepository.
func NewLecturerRepository(db *sql.DB) LecturerRepository {
	return &lecturerRepo{db: db}
}

// ListAllLecturers mengambil semua Dosen dari tabel 'lecturers' (dan join ke tabel 'users' untuk user ID).
func (r *lecturerRepo) ListAllLecturers(ctx context.Context) ([]model.Lecturer, error) {
	query := `
		SELECT 
			l.id, l.user_id, l.lecturer_id, l.department, l.created_at
		FROM 
			lecturers l
		ORDER BY 
			l.lecturer_id;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lecturers []model.Lecturer
	for rows.Next() {
		var l model.Lecturer
		if err := rows.Scan(&l.ID, &l.UserID, &l.LecturerID, &l.Department, &l.CreatedAt); err != nil {
			return nil, err
		}
		lecturers = append(lecturers, l)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lecturers, nil
}

// FindLecturerByID mencari Dosen berdasarkan ID (uuid).
func (r *lecturerRepo) FindLecturerByID(ctx context.Context, id uuid.UUID) (*model.Lecturer, error) {
	query := `
		SELECT 
			id, user_id, lecturer_id, department, created_at
		FROM 
			lecturers
		WHERE 
			id = $1;
	`
	var l model.Lecturer
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&l.ID, &l.UserID, &l.LecturerID, &l.Department, &l.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("lecturer not found")
		}
		return nil, err
	}
	return &l, nil
}

// ListAdviseesByLecturerID mengambil daftar mahasiswa bimbingan.
// ASUMSI: Tabel 'students' memiliki kolom 'advisor_id' yang merujuk ke 'lecturers.id'.
func (r *lecturerRepo) ListAdviseesByLecturerID(ctx context.Context, lecturerID uuid.UUID) ([]model.Advisee, error) {
	query := `
		SELECT 
            s.id, s.student_id, u.full_name, l.department  
        FROM 
            students s
        JOIN 
            users u ON s.user_id = u.id
        JOIN 
            lecturers l ON l.id = s.advisor_id             
        WHERE 
            s.advisor_id = $1
        ORDER BY
            s.student_id;
    `
	rows, err := r.db.QueryContext(ctx, query, lecturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var advisees []model.Advisee
	for rows.Next() {
		var a model.Advisee
		// Scanning ke Advisee model: ID (UUID), StudentID (NIM), Name, Department
		if err := rows.Scan(&a.ID, &a.StudentID, &a.Name, &a.Department); err != nil {
			return nil, err
		}
		advisees = append(advisees, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return advisees, nil
}
// package repository/lecturer.go

func (r *lecturerRepo) FindLecturerByUserID(ctx context.Context, userID uuid.UUID) (*model.Lecturer, error) {
    query := `
        SELECT 
            id, user_id, lecturer_id, department, created_at
        FROM 
            lecturers
        WHERE 
            user_id = $1
        LIMIT 1
    `

    var l model.Lecturer

    err := r.db.QueryRowContext(ctx, query, userID).Scan(
        &l.ID,
        &l.UserID,
        &l.LecturerID,
        &l.Department,
        &l.CreatedAt,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, errors.New("lecturer not found")
        }
        return nil, err
    }

    return &l, nil
}
