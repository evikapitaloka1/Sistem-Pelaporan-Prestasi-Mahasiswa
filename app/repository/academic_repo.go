package repository

import (
    "sistempelaporan/app/model"
    "sistempelaporan/database"
    "database/sql"
)

// --- Students ---

// GET /api/v1/students
func GetAllStudents() ([]map[string]interface{}, error) {
    query := `
        SELECT s.id, s.student_id, s.program_study, s.academic_year, u.full_name, u.email
        FROM students s
        JOIN users u ON s.user_id = u.id
        WHERE u.is_active = true
    `
    rows, err := database.PostgresDB.Query(query)
    if err != nil { return nil, err }
    defer rows.Close()

    var students []map[string]interface{}
    for rows.Next() {
        var id, studentID, prodi, year, name, email string
        if err := rows.Scan(&id, &studentID, &prodi, &year, &name, &email); err != nil { return nil, err }
        
        students = append(students, map[string]interface{}{
            "id": id, "student_id": studentID, "program_study": prodi,
            "academic_year": year, "full_name": name, "email": email,
        })
    }
    return students, nil
}

// GET /api/v1/students/:id
func GetStudentDetail(id string) (map[string]interface{}, error) {
    query := `
        SELECT s.id, s.student_id, s.program_study, s.academic_year, s.advisor_id, 
               u.full_name, u.email, u.username
        FROM students s
        JOIN users u ON s.user_id = u.id
        WHERE s.id = $1
    `
    var s struct {
        ID, StudentID, Prodi, Year string
        AdvisorID *string // Bisa null
        Name, Email, Username string
    }
    
    err := database.PostgresDB.QueryRow(query, id).Scan(
        &s.ID, &s.StudentID, &s.Prodi, &s.Year, &s.AdvisorID, 
        &s.Name, &s.Email, &s.Username,
    )
    if err != nil { return nil, err }

    return map[string]interface{}{
        "id": s.ID, "student_id": s.StudentID, "program_study": s.Prodi,
        "academic_year": s.Year, "advisor_id": s.AdvisorID,
        "user": map[string]string{"full_name": s.Name, "email": s.Email, "username": s.Username},
    }, nil
}

// GET /api/v1/students/:id/achievements
// Mengambil prestasi spesifik milik student tertentu
func GetAchievementsByStudentID(studentID string) ([]model.AchievementReference, error) {
    query := `
        SELECT id, student_id, mongo_achievement_id, status, created_at, submitted_at, verified_at
        FROM achievement_references
        WHERE student_id = $1
        ORDER BY created_at DESC
    `
    rows, err := database.PostgresDB.Query(query, studentID)
    if err != nil { return nil, err }
    defer rows.Close()

    var list []model.AchievementReference
    for rows.Next() {
        var ach model.AchievementReference
        rows.Scan(&ach.ID, &ach.StudentID, &ach.MongoAchievementID, &ach.Status, &ach.CreatedAt, &ach.SubmittedAt, &ach.VerifiedAt)
        list = append(list, ach)
    }
    return list, nil
}

// --- Lecturers ---

// GET /api/v1/lecturers
func GetAllLecturers() ([]map[string]interface{}, error) {
    query := `
        SELECT l.id, l.lecturer_id, l.department, u.full_name, u.email
        FROM lecturers l
        JOIN users u ON l.user_id = u.id
        WHERE u.is_active = true
    `
    rows, err := database.PostgresDB.Query(query)
    if err != nil { return nil, err }
    defer rows.Close()

    var lecturers []map[string]interface{}
    for rows.Next() {
        var id, lecID, dept, name, email string
        rows.Scan(&id, &lecID, &dept, &name, &email)
        lecturers = append(lecturers, map[string]interface{}{
            "id": id, "lecturer_id": lecID, "department": dept, "full_name": name, "email": email,
        })
    }
    return lecturers, nil
}

// GET /api/v1/lecturers/:id/advisees
// Mengambil daftar mahasiswa yang dibimbing oleh dosen tertentu
func GetLecturerAdvisees(lecturerID string) ([]map[string]interface{}, error) {
    query := `
        SELECT s.id, s.student_id, s.program_study, s.academic_year, u.full_name
        FROM students s
        JOIN users u ON s.user_id = u.id
        WHERE s.advisor_id = $1 AND u.is_active = true
    `
    rows, err := database.PostgresDB.Query(query, lecturerID)
    if err != nil { return nil, err }
    defer rows.Close()

    // KOREKSI KRITIS: Inisialisasi sebagai slice kosong (bukan nil)
    advisees := make([]map[string]interface{}, 0) 
    
    for rows.Next() {
        var id, stdID, prodi, year, name string
        if err := rows.Scan(&id, &stdID, &prodi, &year, &name); err != nil { 
            // Jika ada error scanning, kita harus memastikan defer rows.Close() dipanggil
            // dan kita kembalikan error.
            return nil, err
        }
        
        advisees = append(advisees, map[string]interface{}{
            "id": id, "student_id": stdID, "program_study": prodi, 
            "academic_year": year, "full_name": name,
        })
    }
    
    // Pastikan rows.Err() diperiksa setelah loop
    if rows.Err() != nil {
        return nil, rows.Err()
    }
    
    // Jika tidak ada data, ia mengembalikan slice kosong, yang di-marshal ke JSON sebagai []
    return advisees, nil 
}
func GetStudentIDByUserID(userID string) (string, error) {
    query := `SELECT id FROM students WHERE user_id = $1`
    var studentID string
    
    err := database.PostgresDB.QueryRow(query, userID).Scan(&studentID)
    
    if err == sql.ErrNoRows {
        return "", nil // User bukan mahasiswa
    }
    return studentID, err
}

// Menangani error: undefined: repository.ExtractAdvisorID
// Fungsi helper untuk mendapatkan Advisor ID dari hasil GetStudentDetail (map)
func ExtractAdvisorID(student map[string]interface{}) string {
    var advisorID string
    if idRaw, ok := student["advisor_id"]; ok && idRaw != nil {
        if idPtr, isPtr := idRaw.(*string); isPtr && idPtr != nil {
            advisorID = *idPtr
        } else if idStr, isStr := idRaw.(string); isStr {
            advisorID = idStr
        }
    }
    return advisorID
}