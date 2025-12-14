package route

import (
    "sistempelaporan/app/service"
    "sistempelaporan/middleware" // Pastikan package middleware sudah ada

    "github.com/gofiber/fiber/v2"
)


func AcademicRoutes(router fiber.Router) {
    // --- Group Students ---
    // Semua endpoint di bawah ini butuh Login (JWT)
    students := router.Group("/students", middleware.Protected())
    
    // FR-View Student List
    students.Get("/", service.GetStudents)
    
    // FR-View Student Detail
    students.Get("/:id", service.GetStudentByID)
    
    // FR-View Student Achievements
    students.Get("/:id/achievements", service.GetStudentAchievements)
    
    // FR-Assign Advisor (Hanya Admin yang boleh)
    // Sesuai FR-009: Admin manage users/profiles
    students.Put("/:id/advisor", middleware.CheckPermission("Admin"), service.UpdateStudentAdvisor)

    // --- Group Lecturers ---
    // Semua endpoint di bawah ini butuh Login (JWT)
    lecturers := router.Group("/lecturers", middleware.Protected())
    
    // FR-View Lecturer List
    lecturers.Get("/", service.GetLecturers)
    
    // FR-View Advisees (Mahasiswa Bimbingan)
    lecturers.Get("/:id/advisees", service.GetLecturerAdvisees)
}