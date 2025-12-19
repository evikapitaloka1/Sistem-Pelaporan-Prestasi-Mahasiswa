package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func AchievementRoutes(r fiber.Router) {
    // Group ini sudah diproteksi oleh JWT
    ach := r.Group("/achievements", middleware.Protected())
    ach.Get("/", middleware.CheckPermission("achievement:read"), service.GetListAchievements)
    ach.Get("/:id", middleware.CheckPermission("achievement:read"), middleware.AuthorizeResource("student_read"), service.GetAchievementDetail)
    ach.Post("/", middleware.CheckPermission("achievement:create"), service.SubmitAchievement)
    ach.Put("/:id", middleware.CheckPermission("achievement:update"), middleware.AuthorizeResource("student_read"), service.UpdateAchievement)
    ach.Delete("/:id", middleware.CheckPermission("achievement:delete"),middleware.AuthorizeResource("student_read"), service.DeleteAchievement)
    ach.Post("/:id/submit", middleware.CheckPermission("achievement:submit"), middleware.AuthorizeResource("student_read"), service.RequestVerification)
    ach.Post("/:id/verify", middleware.CheckPermission("achievement:verify"), middleware.AuthorizeResource("student_read"), service.VerifyAchievement)
    ach.Post("/:id/reject", middleware.CheckPermission("achievement:reject"), middleware.AuthorizeResource("student_read"), service.RejectAchievement)
    ach.Post("/:id/attachments", middleware.CheckPermission("achievement:upload"), middleware.AuthorizeResource("student_read"), service.UploadAttachment)
    ach.Get("/:id/history", middleware.CheckPermission("achievement:read"), middleware.AuthorizeResource("student_read"), service.GetHistory)
}