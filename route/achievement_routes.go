package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func AchievementRoutes(r fiber.Router) {
    // Group ini sudah diproteksi oleh JWT
    ach := r.Group("/achievements", middleware.Protected())

    // 1. GET List (Filtered di service)
    ach.Get("/", middleware.CheckPermission("achievement:read"), service.GetListAchievements)

    // 2. GET Detail (Butuh akses baca & cek kepemilikan)
    ach.Get("/:id", middleware.CheckPermission("achievement:read"), middleware.AuthorizeResource("student_read"), service.GetAchievementDetail)

    // 3. POST Create (Mahasiswa/Admin)
    ach.Post("/", middleware.CheckPermission("achievement:create"), service.SubmitAchievement)

    // 4. PUT Update Content (Edit Draft)
    ach.Put("/:id", middleware.CheckPermission("achievement:update"), middleware.AuthorizeResource("student_read"), service.UpdateAchievement)

    // 5. DELETE Achievement
    ach.Delete("/:id", middleware.CheckPermission("achievement:delete"), service.DeleteAchievement)

    // 6. POST Submit Verification (Ubah ke Status Submitted)
    // Sekarang menggunakan permission spesifik: achievement:submit
    ach.Post("/:id/submit", middleware.CheckPermission("achievement:submit"), middleware.AuthorizeResource("student_read"), service.RequestVerification)

    // 7. POST Verify (Dosen Wali Approve)
    // Sekarang menggunakan permission spesifik: achievement:verify
    ach.Post("/:id/verify", middleware.CheckPermission("achievement:verify"), middleware.AuthorizeResource("student_read"), service.VerifyAchievement)

    // 8. POST Reject (Dosen Wali Reject)
    // Sekarang menggunakan permission spesifik: achievement:reject
    ach.Post("/:id/reject", middleware.CheckPermission("achievement:reject"), middleware.AuthorizeResource("student_read"), service.RejectAchievement)

    // 9. POST Upload Attachments
    // Sekarang menggunakan permission spesifik: achievement:upload
    ach.Post("/:id/attachments", middleware.CheckPermission("achievement:upload"), middleware.AuthorizeResource("student_read"), service.UploadAttachment)
    
    // 10. GET History
    ach.Get("/:id/history", middleware.CheckPermission("achievement:read"), middleware.AuthorizeResource("student_read"), service.GetHistory)
}