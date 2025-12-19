package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func ReportRoutes(router fiber.Router) {
    reports := router.Group("/reports", middleware.Protected())

    reports.Get("/statistics", 
        middleware.CheckPermission("achievement:read"), 
        service.GetGeneralStatistics)
    reports.Get("/student/:id", 
        middleware.CheckPermission("achievement:read"),
        middleware.AuthorizeResource("student_read"), 
        service.GetStudentReport)
}