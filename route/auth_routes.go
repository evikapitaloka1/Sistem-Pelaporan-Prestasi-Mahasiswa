package route

import (
	"sistempelaporan/app/service"
	"sistempelaporan/middleware"

	"github.com/gofiber/fiber/v2"
)

func AuthRoutes(r fiber.Router) {

	auth := r.Group("/auth")
	auth.Post("/refresh", service.RefreshToken)
	auth.Post("/logout", middleware.Protected(), service.Logout)
	auth.Get("/profile", middleware.Protected(), service.GetProfile)
}