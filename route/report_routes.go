package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

// ReportRoutes mendaftarkan endpoint reporting
func ReportRoutes(router fiber.Router) {
	// Semua route di group ini diproteksi oleh JWT
	reports := router.Group("/reports", middleware.Protected())

	// Statistik Global (Dashboard Admin/Dosen)
	// GET /api/v1/reports/statistics
	// Otorisasi: Admin (user:manage) atau Dosen (asumsi memiliki permission baca statistik)
	reports.Get("/statistics", 
        // Menggunakan CheckPermission, asumsi Admin dan Dosen memiliki 'report:read_stats'
        // Jika hanya Admin yang boleh, gunakan CheckPermission("user:manage")
        middleware.CheckPermission("user:manage"), 
        service.GetGeneralStatistics)

	// Laporan Spesifik Mahasiswa
	// GET /api/v1/reports/student/:id
	// Otorisasi: Admin, Mahasiswa Ybs, Dosen Wali (Menggunakan middleware relasional)
	reports.Get("/student/:id", 
        // Middleware Protected() sudah ada di group, jadi tidak perlu diulang.
        middleware.AuthorizeResource("student_read"), // <-- Otorisasi 3-Tingkat
        service.GetStudentReport)
}