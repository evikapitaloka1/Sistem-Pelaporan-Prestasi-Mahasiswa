package service

import (
	"sync"
	"sistempelaporan/app/repository"
	"sistempelaporan/helper"  // Untuk fungsi perhitungan jika ada
    "strings"   // WAJIB: Untuk strings.EqualFold     // Untuk WaitGroup
    "time"      // WAJIB: Untuk time.Now()
	"github.com/gofiber/fiber/v2"
)

// GetGeneralStatistics godoc
// @Summary      Statistik Prestasi Umum
// @Description  Menghasilkan statistik prestasi yang mencakup top mahasiswa, tren bulanan (Postgres), dan distribusi tipe prestasi (MongoDB).
// @Tags         Reports & Analytics
// @Produce      json
// @Success      200  {object}  helper.Response
// @Failure      500  {object}  helper.Response
// @Router       /reports/statistics [get]
// @Security     BearerAuth
func GetGeneralStatistics(c *fiber.Ctx) error {
    // 1. Identifikasi User dari Context (Set oleh middleware Protected)
    userID := c.Locals("user_id").(string)
    role := c.Locals("role").(string)

    // Variabel filter untuk repository
    var targetID string // Akan berisi StudentID atau LecturerID tergantung role
    
    // 2. Logika Penentuan Filter (Aktor FR-011)
    if strings.EqualFold(role, "Mahasiswa") {
        // Actor: Mahasiswa (own) -> Ambil Student ID milik user ini
        mhsID, err := repository.GetStudentIDByUserID(userID)
        if err != nil {
            return helper.Error(c, fiber.StatusNotFound, "Data mahasiswa tidak ditemukan", nil)
        }
        targetID = mhsID
    } else if strings.EqualFold(role, "Dosen Wali") {
        // Actor: Dosen Wali (advisee) -> Ambil Lecturer ID milik user ini
        lecturerID, err := repository.GetLecturerIDByUserID(userID)
        if err != nil {
            return helper.Error(c, fiber.StatusNotFound, "Data dosen tidak ditemukan", nil)
        }
        targetID = lecturerID
    }
    // Note: Jika role "Admin" (all), targetID tetap kosong agar repository menarik semua data

    var wg sync.WaitGroup
    
    // Variabel penampung hasil
    var topStudents []map[string]interface{}
    var monthlyTrend []map[string]interface{}
    var typeDist     []map[string]interface{}
    
    // Variabel error
    var err1, err2, err3 error

    // 3. Eksekusi 3 Lapisan Statistik secara Paralel (FR-011 Output)
    wg.Add(3) 

    // Tugas 1: Top Mahasiswa Berprestasi (Postgres)
    go func() {
        defer wg.Done()
        topStudents, err1 = repository.GetTopStudentsStats(targetID, role)
    }()

    // Tugas 2: Total Prestasi per Periode / Monthly Trend (Postgres)
    go func() {
        defer wg.Done()
        monthlyTrend, err2 = repository.GetMonthlyTrendStats(targetID, role)
    }()

    // Tugas 3: Total per Tipe & Distribusi Tingkat Kompetisi (Mongo)
    go func() {
        defer wg.Done()
        typeDist, err3 = repository.GetAchievementTypeDistribution(targetID, role)
    }()

    // Tunggu semua goroutine selesai
    wg.Wait()

    // 4. Error Handling
    if err1 != nil || err2 != nil || err3 != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengolah statistik laporan", nil)
    }

    // 5. Gabungkan Response JSON
    response := fiber.Map{
        "top_students":      topStudents,    // Output: Top mahasiswa
        "monthly_trend":     monthlyTrend,   // Output: Total per periode
        "type_distribution": typeDist,       // Output: Per tipe & Tingkat kompetisi
        "generated_at":      time.Now().Format("2006-01-02 15:04:05"),
        "reporting_scope":   role,           // Memberi tahu user cakupan data yang muncul
    }

    return helper.Success(c, response, "Statistik prestasi berhasil dibuat")
}

// GetStudentReport godoc
// @Summary      Laporan Prestasi per Mahasiswa
// @Description  Mengambil laporan mendalam untuk satu mahasiswa tertentu, termasuk profil dan daftar lengkap prestasi mereka.
// @Tags         Reports & Analytics
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  helper.Response
// @Failure      404  {object}  helper.Response
// @Failure      500  {object}  helper.Response
// @Router       /reports/student/{id} [get]
// @Security     BearerAuth
func GetStudentReport(c *fiber.Ctx) error {
	studentID := c.Params("id")

	// 1. Ambil Profil Mahasiswa
	student, err := repository.GetStudentDetail(studentID)
	if err != nil {
		return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
	}

	// 2. Ambil List Prestasi
	achievements, err := repository.GetAchievementsByStudentID(studentID)
	if err != nil {
		return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data prestasi", err.Error())
	}

	response := fiber.Map{
		"student_profile":    student,
		"total_achievements": len(achievements),
		"achievements_list":  achievements,
	}

	return helper.Success(c, response, "Laporan prestasi mahasiswa berhasil diambil")
}