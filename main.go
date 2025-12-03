package main

import (
	"log"
	
	// Import library untuk memuat file .env
	"github.com/joho/godotenv" 
	
	"github.com/gofiber/fiber/v2"
	
	db "uas/database/postgres"
	mongodb "uas/database/mongo"

	postgresRoutes "uas/route/postgres"
	mongoRoutes "uas/route/mongo"
)

func main() {
	// ----------------------------------------------------
	// FIX: Muat Variabel Lingkungan Global (.env)
	// ----------------------------------------------------
	if err := godotenv.Load(); err != nil {
		log.Println("WARNING: Could not load .env file. Relying on system environment variables.")
	}
	log.Println("Environment variables loaded.")
	// ----------------------------------------------------
	
	// 1. Inisialisasi koneksi (Sekarang sudah bisa menemukan DSN dari ENV)
	db.Connect()
	mongodb.Connect()

	// 2. Ambil instance koneksi DB PostgreSQL
	postgreDB := db.GetDB() 

	// 3. Ambil instance koneksi MongoDB (*mongo.Client)
	mongoClient := mongodb.GetClient()

	app := fiber.New()

	// 4. Daftarkan Route PostgreSQL
	postgresRoutes.RegisterRoutes(app, postgreDB, mongoClient)

	// 5. Ambil collection MongoDB untuk route Mongo
	achievementColl := mongodb.GetCollection("uas", "achievements")
	mongoRoutes.RegisterRoutesMongo(app, achievementColl, postgreDB)
	
	// 6. Jalankan Aplikasi
	log.Println("Fiber server starting on :8080")
	if err := app.Listen(":8080"); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}