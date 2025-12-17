package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func AchievementRoutes(r fiber.Router) {
	// Middleware Protected dipasang di group ini (Wajib Login)
	ach := r.Group("/achievements", middleware.Protected())

	// 1. GET List (Filtered)
	ach.Get("/", middleware.CheckPermission("achievement:read"), service.GetListAchievements)

	// 2. GET Detail (:id)
	ach.Get("/:id", middleware.CheckPermission("achievement:read"), service.GetAchievementDetail)

	// 3. POST Create (Mahasiswa)
	ach.Post("/", middleware.CheckPermission("achievement:create"), service.SubmitAchievement)

	// 4. PUT Update (:id) - Edit Content
	ach.Put("/:id", middleware.CheckPermission("achievement:update"), service.UpdateAchievement)

	// 5. DELETE (:id)
	ach.Delete("/:id", middleware.CheckPermission("achievement:delete"), service.DeleteAchievement)

	// 6. POST Submit Verification (:id/submit)
	ach.Post("/:id/submit", middleware.CheckPermission("achievement:update"), service.RequestVerification)

	// 7. POST Verify (Dosen Wali Approve)
	ach.Post("/:id/verify", middleware.CheckPermission("achievement:verify"), service.VerifyAchievement)

	// 8. POST Reject (Dosen Wali Reject)
	ach.Post("/:id/reject", middleware.CheckPermission("achievement:reject"), service.RejectAchievement)

	// 9. POST Upload Attachments
	ach.Post("/:id/attachments", middleware.CheckPermission("achievement:update"), service.UploadAttachment)
	
	// 10. GET History
	ach.Get("/:id/history", middleware.AuthorizeResource("student_read"), service.GetHistory)
}