package route

import (
    "sistempelaporan/app/service"
    "sistempelaporan/middleware" // Pastikan package middleware sudah ada

    "github.com/gofiber/fiber/v2"
)


func AcademicRoutes(router fiber.Router) {
	// ... (imports)
    readAllAccess := middleware.CheckPermission("user:read_all") 

	// Group Students
	students := router.Group("/students", middleware.Protected())
	
	// FR-View Student List (Admin/Dosen)
	students.Get("/", middleware.AuthorizeResource("student_read"), service.GetStudents)

	// FR-View Student Detail (Otorisasi 3-Tingkat)
	students.Get("/:id", middleware.AuthorizeResource("student_read"), service.GetStudentByID)

	// FR-View Student Achievements (Otorisasi 3-Tingkat)
	students.Get("/:id/achievements", middleware.AuthorizeResource("student_read"), service.GetStudentAchievements)

	// FR-Assign Advisor (Admin Only)
	students.Put("/:id/advisor", middleware.CheckPermission("user:manage"), service.UpdateStudentAdvisor)

	// Group Lecturers
	lecturers := router.Group("/lecturers", middleware.Protected())
	
	// FR-View Lecturer List (Admin/Dosen)
	lecturers.Get("/", readAllAccess, service.GetLecturers)
	
	// FR-View Advisees (Self-Access Dosen)
	lecturers.Get("/:id/advisees", middleware.CanAccessSelf(), service.GetLecturerAdvisees)
}