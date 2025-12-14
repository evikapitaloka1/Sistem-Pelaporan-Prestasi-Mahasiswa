package route

import (
    "sistempelaporan/app/service"
    "sistempelaporan/middleware"

    "github.com/gofiber/fiber/v2"
)

// ReportRoutes mendaftarkan endpoint reporting
// Sesuai SRS 
func ReportRoutes(router fiber.Router) {
    reports := router.Group("/reports", middleware.Protected())

    // Statistik Global (Dashboard Admin/Dosen)
    // GET /api/v1/reports/statistics
    reports.Get("/statistics", service.GetGeneralStatistics)

    // Laporan Spesifik Mahasiswa
    // GET /api/v1/reports/student/:id
    reports.Get("/student/:id", service.GetStudentReport)
}