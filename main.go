package main

import (
	"log"
	"os"

	"sistempelaporan/database"
	
	// [PERBAIKAN 1] Import sesuai nama folder di foto ('route')
	"sistempelaporan/route"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

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

	// 5. SETUP ROUTES
	// [PERBAIKAN 2] Panggil dengan 'route.' (tanpa 's') sesuai nama folder
	route.SetupRoutes(app)

	// 6. Start Server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("🚀 Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}