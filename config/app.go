package config

import (
	"sistempelaporan/route"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// NewFiberApp menginisialisasi Fiber, Middleware, dan Routes
func NewFiberApp() *fiber.App {
	// 1. Init Fiber App dengan Custom Error Handler
	app := fiber.New(fiber.Config{
		AppName: "Sistem Pelaporan Prestasi Mahasiswa API",
		// Global Error Handler: Menangkap error yang tidak ter-handle di controller
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default error code 500
			code := fiber.StatusInternalServerError
			
			// Jika error berasal dari Fiber (misal 404 Not Found), gunakan codenya
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

	// 2. Pasang Middleware Global
	app.Use(cors.New())                    // Agar bisa diakses dari Frontend beda domain
	app.Use(logger.New(NewLoggerConfig())) // Menggunakan config logger dari file logger.go
	app.Use(recover.New())                 // Agar server tidak crash total jika ada panic (critical error)

	// 3. Daftarkan Semua Route
	// Ini memanggil fungsi SetupRoutes dari folder route/index.go
	route.SetupRoutes(app)

	return app
}