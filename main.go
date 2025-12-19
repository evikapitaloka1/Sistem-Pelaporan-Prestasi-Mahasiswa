package main

import (
	"log"
	"os"

	"sistempelaporan/database"
	"sistempelaporan/route"
    
	// [TAMBAHAN 1] Import folder docs yang akan di-generate oleh swag
	_ "sistempelaporan/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
    

	// [TAMBAHAN 2] Import middleware swagger untuk Fiber
	// Ganti alias 'swagger' menjadi 'fiberSwagger' agar tidak bentrok
swagger "github.com/swaggo/fiber-swagger"
)

// [TAMBAHAN 3] ANOTASI GLOBAL (Sesuai Modul 11 hal. 3 & SRS kamu)
// @title Sistem Pelaporan Prestasi Mahasiswa API
// @version 1.0
// @description API Backend untuk manajemen pelaporan prestasi mahasiswa, verifikasi dosen wali, dan admin. [cite: 6, 409]
// @host localhost:3000
// @BasePath /api/v1
// @schemes http

// ================= SECURITY JWT =================
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token dengan format -> Bearer <token>
// ===============================================
func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using system environment variables")
	}

	// 2. Connect Databases
	database.ConnectPostgres()
	database.ConnectMongo()

	// 3. Init Fiber App
	app := fiber.New(fiber.Config{
		AppName: "Sistem Pelaporan Prestasi Mahasiswa API",
	})

	// 4. Middlewares Global
	app.Use(logger.New())
	app.Use(cors.New())

	// [TAMBAHAN 4] ROUTE SWAGGER (Sesuai Modul 11 hal. 7 & 12)
	// Akses di: http://localhost:3000/swagger/index.html
	// Gunakan alias yang baru saja dibuat
	app.Get("/swagger/*", swagger.WrapHandler)

	// 5. SETUP ROUTES
	route.SetupRoutes(app)

	// 6. Start Server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("🚀 Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}