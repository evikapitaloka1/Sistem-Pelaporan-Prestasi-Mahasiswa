package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(r fiber.Router) {
	// Grouping URL: /api/v1/auth
	auth := r.Group("/auth")

	// 1. Login
	// Memanggil service.Login yang menerima *fiber.Ctx (Sesuai Modul 4)
	auth.Post("/login", service.Login)

	// 2. Refresh Token
	auth.Post("/refresh", service.RefreshToken)

	// 3. Logout (Butuh Token)
	// Middleware Protected dipasang sebelum memanggil service
	auth.Post("/logout", middleware.Protected(), service.Logout)

	// 4. Get Profile (Butuh Token)
	auth.Get("/profile", middleware.Protected(), service.GetProfile)
}