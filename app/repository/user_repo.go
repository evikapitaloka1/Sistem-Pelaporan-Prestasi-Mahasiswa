package repository

import (
	"context"
	"database/sql"
	"errors"
	"sistempelaporan/app/model"
	"sistempelaporan/database"
	"time"
	"sync"
	"strings"
)

// --- Login & Auth ---

func FindUserByUsername(username string) (*model.User, error) {
	// Query join ke tabel roles untuk ambil nama role sekalian
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.role_id, r.name 
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username = $1 AND u.is_active = true
	`

	var user model.User
	var roleName string

	// Scan data dari database
	row := database.PostgresDB.QueryRow(query, username)
	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
		&user.FullName, &user.RoleID, &roleName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	user.Role.Name = roleName
	return &user, nil
}
// [BARU] Untuk endpoint GET /auth/profile
// File: app/repository/user_repository.go

func FindUserByID(id string) (*model.User, error) {
    query := `
        SELECT u.id, u.username, u.email, u.full_name, u.role_id, 
               u.is_active, u.created_at, u.updated_at,
               r.name, r.description     -- <-- FIX: Tambahkan r.description
        FROM users u
        JOIN roles r ON u.role_id = r.id
        WHERE u.id = $1
    `

    var user model.User
    var roleName string
    var roleDescription string // <-- Variabel baru untuk menampung deskripsi

    row := database.PostgresDB.QueryRow(query, id)
    err := row.Scan(
        &user.ID, 
        &user.Username, 
        &user.Email, 
        &user.FullName, 
        &user.RoleID, 
        &user.IsActive, 
        &user.CreatedAt, 
        &user.UpdatedAt, 
        &roleName,
        &roleDescription,    // <-- Scan deskripsi
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, errors.New("user not found")
        }
        return nil, err
    }

    user.Role.ID = user.RoleID
    user.Role.Name = roleName
    user.Role.Description = roleDescription // <-- Set Description
    
    return &user, nil
}

func GetPermissionsByRoleID(roleID string) ([]string, error) {
	query := `
		SELECT p.name 
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`

	rows, err := database.PostgresDB.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}
// --- Helper Identitas ---

func FindStudentByUserID(userID string) (*model.Student, error) {
	query := `SELECT id, student_id, program_study, advisor_id FROM students WHERE user_id = $1`
	var s model.Student
	err := database.PostgresDB.QueryRow(query, userID).Scan(&s.ID, &s.StudentID, &s.ProgramStudy, &s.AdvisorID)
	if err != nil {
		return nil, err
	}
	s.UserID.UnmarshalText([]byte(userID))
	return &s, nil
}

func FindLecturerByUserID(userID string) (*model.Lecturer, error) {
	query := `SELECT id, lecturer_id, department FROM lecturers WHERE user_id = $1`
	var l model.Lecturer
	err := database.PostgresDB.QueryRow(query, userID).Scan(&l.ID, &l.LecturerID, &l.Department)
	if err != nil {
		return nil, err
	}
	l.UserID.UnmarshalText([]byte(userID))
	return &l, nil
}

// --- FR-009: Manage Users (Transaction) ---

func CreateUserWithProfile(user *model.User, student *model.Student, lecturer *model.Lecturer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := database.PostgresDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// A. Insert ke tabel Users
	userQuery := `
		INSERT INTO users (id, username, email, password_hash, full_name, role_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, userQuery, 
		user.ID, user.Username, user.Email, user.PasswordHash, user.FullName, user.RoleID, true, time.Now(),
	).Scan(&user.ID)

	if err != nil {
        // 💡 FIX 2.1: Pemeriksaan error Foreign Key spesifik
        if strings.Contains(err.Error(), "users_role_id_fkey") {
            // Kita kembalikan error yang lebih jelas agar service bisa menangkapnya
            return errors.New("Role ID yang dimasukkan tidak ditemukan di tabel roles.")
        }
        // 💡 FIX 2.2: Pemeriksaan error NOT NULL (jika password_hash kosong)
        if strings.Contains(err.Error(), "violates not-null constraint") {
            return errors.New("Kolom wajib (NOT NULL) kosong, periksa password, username, atau email.")
        }
        
		return err
	}

	// B. Insert Student
	if student != nil {
		studentQuery := `
			INSERT INTO students (id, user_id, student_id, program_study, academic_year, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = tx.ExecContext(ctx, studentQuery, 
			student.ID, user.ID, student.StudentID, student.ProgramStudy, student.AcademicYear, time.Now(),
		)
		if err != nil {
			return err
		}
	}

	// C. Insert Lecturer
	if lecturer != nil {
		// ... (kode insert lecturer)
	}

	return tx.Commit()
}

func AssignAdvisorToStudent(studentID string, advisorID string) error {
	query := `UPDATE students SET advisor_id = $1 WHERE id = $2`
	_, err := database.PostgresDB.Exec(query, advisorID, studentID)
	return err
}
// --- FR-009: Tambahan CRUD User (Admin) ---

// 1. Mengambil Semua User (GET /api/v1/users)
// File: app/repository/user_repository.go

// 1. Mengambil Semua User (GET /api/v1/users)
// File: app/repository/user_repository.go

// File: app/repository/user_repository.go

func FindAllUsers() ([]model.User, error) {
    query := `
        SELECT u.id, u.username, u.email, u.full_name, u.role_id, r.name, 
               u.is_active, u.created_at, u.updated_at, r.description -- <-- FIX: Tambahkan semua field
        FROM users u
        JOIN roles r ON u.role_id = r.id
        WHERE u.is_active = true
        ORDER BY u.created_at DESC
    `

    rows, err := database.PostgresDB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var users []model.User
    for rows.Next() {
        var user model.User
        var roleName string
        var roleDescription string // <-- Variabel baru untuk menampung deskripsi
        
        // Scan data per baris
        err := rows.Scan(
            &user.ID, 
            &user.Username, 
            &user.Email, 
            &user.FullName, 
            &user.RoleID, 
            &roleName,
            &user.IsActive,   // <-- FIX: Scan IsActive
            &user.CreatedAt,  // <-- FIX: Scan CreatedAt
            &user.UpdatedAt,  // <-- FIX: Scan UpdatedAt
            &roleDescription, // <-- FIX: Scan Description
        )
        if err != nil {
            return nil, err
        }

        user.Role.ID = user.RoleID
        user.Role.Name = roleName
        user.Role.Description = roleDescription // <-- Set Description
        users = append(users, user)
    }

    return users, nil
}
// 2. Update Data User General (PUT /api/v1/users/:id)
// Hanya update nama, username, email. Password biasanya dipisah endpointnya.
func UpdateUserGeneral(id string, username, fullName, email string) error {
    query := `
        UPDATE users 
        SET username = $1, full_name = $2, email = $3, updated_at = NOW()
        WHERE id = $4
    `
    _, err := database.PostgresDB.Exec(query, username, fullName, email, id)
    return err
}

// 3. Update Role User (PUT /api/v1/users/:id/role)
func UpdateUserRole(id string, roleID string) error {
    query := `
        UPDATE users 
        SET role_id = $1, updated_at = NOW()
        WHERE id = $2
    `
    _, err := database.PostgresDB.Exec(query, roleID, id)
    return err
}

// 4. Soft Delete User (DELETE /api/v1/users/:id)
// Kita tidak menghapus baris, tapi set is_active = false agar data relasi aman
func DeleteUserByID(id string) error {
    query := `
        UPDATE users 
        SET is_active = false, updated_at = NOW()
        WHERE id = $1
    `
    _, err := database.PostgresDB.Exec(query, id)
    return err
}
// =======================================================
// BLACKLIST (SIMULASI REDIS/CACHE)
// =======================================================

// tokenBlacklist: Menyimpan token yang sudah di-logout
var tokenBlacklist = make(map[string]time.Time)
var mutex sync.RWMutex

// SetTokenBlacklist mencatat token ke blacklist dengan TTL
// Fungsi ini DIEKSPOR karena dimulai dengan huruf kapital (S)
func SetTokenBlacklist(token string, ttl time.Duration) error {
	mutex.Lock()
	defer mutex.Unlock()
	
	// Set waktu token akan 'kadaluarsa' dari blacklist.
	tokenBlacklist[token] = time.Now().Add(ttl)
	
	return nil
}

// IsTokenBlacklisted mengecek apakah token ada di blacklist dan masih aktif
// Fungsi ini DIEKSPOR karena dimulai dengan huruf kapital (I)
func IsTokenBlacklisted(token string) bool {
	mutex.RLock()
	defer mutex.RUnlock()

	expTime, found := tokenBlacklist[token]
	if !found {
		return false // Token tidak ada di Blacklist
	}

	// Cek apakah token Blacklist-nya sendiri belum kadaluarsa
	if time.Now().Before(expTime) {
		return true // Token sudah di-logout (Blacklisted)
	}

	// Jika sudah lewat expTime, tokenBlacklist bisa dihapus (namun ini memerlukan goroutine)
	return false // Token dianggap sudah tidak di Blacklist
}
