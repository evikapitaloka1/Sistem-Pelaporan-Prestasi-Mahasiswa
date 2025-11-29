package routes

import (
	repository "uas/app/repository/postgres"
	service "uas/app/service/postgres"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App) {

	api := app.Group("/api/v1")

	// ===== Repositori & Service =====
	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(userRepo)

	authRepo := repository.NewAuthRepository()
	authService := service.NewAuthService(authRepo)

	// ===== User Routes =====
	SetupUserRoutes(api, userService, authService) // kirim juga authService

	// ===== Auth Routes =====
	SetupAuthRoutes(api, authService)
}
