package config

import (
	"sistempelaporan/route"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)


func NewFiberApp() *fiber.App {
	
	app := fiber.New(fiber.Config{
		AppName: "Sistem Pelaporan Prestasi Mahasiswa API",
	
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			
			code := fiber.StatusInternalServerError
			
			
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}

			return c.Status(code).JSON(fiber.Map{
				"code":    code,
				"status":  "Error",
				"message": err.Error(),
			})
		},
	})

	
	app.Use(cors.New())                    
	app.Use(logger.New(NewLoggerConfig())) 
	app.Use(recover.New())                 
	route.SetupRoutes(app)

	return app
}