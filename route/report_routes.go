package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

// ReportRoutes mendaftarkan endpoint reporting
func ReportRoutes(router fiber.Router) {
    reports := router.Group("/reports", middleware.Protected())

    // Perbaikan: Ubah permission agar Mahasiswa & Dosen juga bisa masuk
    // Kita gunakan "achievement:read" yang dimiliki semua aktor tersebut
    reports.Get("/statistics", 
        middleware.CheckPermission("achievement:read"), 
        service.GetGeneralStatistics)

    reports.Get("/student/:id", 
        middleware.CheckPermission("achievement:read"),
        middleware.AuthorizeResource("student_read"), 
        service.GetStudentReport)
}