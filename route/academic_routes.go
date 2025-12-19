package route

import (
    "sistempelaporan/app/service"
    "sistempelaporan/middleware" // Pastikan package middleware sudah ada

    "github.com/gofiber/fiber/v2"
)


func AcademicRoutes(router fiber.Router) {
	
    readAllAccess := middleware.CheckPermission("user:read_all") 
	students := router.Group("/students", middleware.Protected())
	students.Get("/", middleware.AuthorizeResource("student_read"), service.GetStudents)
	students.Get("/:id", middleware.AuthorizeResource("student_read"), service.GetStudentByID)
	students.Get("/:id/achievements", middleware.AuthorizeResource("student_read"), service.GetStudentAchievements)
	students.Put("/:id/advisor", middleware.CheckPermission("user:manage"), service.UpdateStudentAdvisor)
	lecturers := router.Group("/lecturers", middleware.Protected())
	lecturers.Get("/", readAllAccess, service.GetLecturers)
	lecturers.Get("/:id/advisees", middleware.CanAccessSelf(), service.GetLecturerAdvisees)
}