package service

import (
    "sync"
    "sistempelaporan/app/repository"
    "sistempelaporan/helper"

    "github.com/gofiber/fiber/v2"
)

// FR-011: Achievement Statistics
// GET /api/v1/reports/statistics
func GetGeneralStatistics(c *fiber.Ctx) error {
    var wg sync.WaitGroup
    
    // Variabel penampung hasil
    var topStudents []map[string]interface{}
    var monthlyTrend []map[string]interface{}
    var typeDist []map[string]interface{}
    
    // Variabel error
    var err1, err2, err3 error

    wg.Add(3) // Kita akan menjalankan 3 tugas sekaligus

    // Tugas 1: Ambil Top Students (Postgres)
    go func() {
        defer wg.Done()
        topStudents, err1 = repository.GetTopStudentsStats()
    }()

    // Tugas 2: Ambil Tren Bulanan (Postgres)
    go func() {
        defer wg.Done()
        monthlyTrend, err2 = repository.GetMonthlyTrendStats()
    }()

    // Tugas 3: Ambil Distribusi Tipe (Mongo)
    go func() {
        defer wg.Done()
        typeDist, err3 = repository.GetAchievementTypeDistribution()
    }()

    // Tunggu semua selesai
    wg.Wait()

    // Cek jika ada error fatal
    if err1 != nil || err2 != nil || err3 != nil {
        return helper.Error(c, fiber.StatusInternalServerError, "Gagal mengolah statistik", nil)
    }

    // Gabungkan response
    response := fiber.Map{
        "top_students":        topStudents,
        "monthly_trend":       monthlyTrend,
        "type_distribution":   typeDist,
        "generated_at":        "now",
    }

    return helper.Success(c, response, "Statistik sistem berhasil diambil")
}

// FR-Baru: Report per Student
// GET /api/v1/reports/student/:id
func GetStudentReport(c *fiber.Ctx) error {
    studentID := c.Params("id")

    // 1. Ambil Profil Mahasiswa (Pakai fungsi repo akademik yg sudah ada)
    student, err := repository.GetStudentDetail(studentID)
    if err != nil {
        return helper.Error(c, fiber.StatusNotFound, "Mahasiswa tidak ditemukan", nil)
    }

    // 2. Ambil List Prestasi (Pakai fungsi repo akademik yg sudah ada)
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