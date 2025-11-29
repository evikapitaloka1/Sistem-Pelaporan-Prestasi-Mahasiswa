package routes

import (
	repository "uas/app/repository/postgres"
	service "uas/app/service/postgres"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(app *fiber.App) {

	api := app.Group("/api/v1")

	// ===== User Routes =====
	userRepo := repository.NewUserRepository()
	userService := service.NewUserService(userRepo)
	SetupUserRoutes(api, userService)

	// ===== Auth Routes =====
	authRepo := repository.NewAuthRepository()
	authService := service.NewAuthService(authRepo)
	SetupAuthRoutes(api, authService)
}
