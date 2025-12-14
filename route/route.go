package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	// 1. Middleware Global
	app.Use(cors.New())
	app.Use(logger.New())

	// 2. Grouping API v1
	api := app.Group("/api/v1")

	// 3. Panggil Route Module Lain
	// Kita passing variable 'api' ke fungsi-fungsi ini agar terdaftar
	AuthRoutes(api)        
	AchievementRoutes(api) 
	AcademicRoutes(api)
	ReportRoutes(api)
	UsersRoutes(api)
}
