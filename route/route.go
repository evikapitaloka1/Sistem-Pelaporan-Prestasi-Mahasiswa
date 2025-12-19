package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	
	app.Use(cors.New())
	app.Use(logger.New())


	api := app.Group("/api/v1")

	AuthRoutes(api)        
	AchievementRoutes(api) 
	AcademicRoutes(api)
	ReportRoutes(api)
	UsersRoutes(api)
}
