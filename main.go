package main

import (
	db "uas/database/postgres"
	mongodb "uas/database/mongo"

	postgresRoutes "uas/route/postgres"
	mongoRoutes "uas/route/mongo"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// 1. Inisialisasi Koneksi
	db.Connect()
	mongodb.Connect()

	// 2. Ambil instance koneksi DB PostgreSQL yang sudah aktif
	postgreDB := db.GetDB() 

	app := fiber.New()

	// 3. Daftarkan Route PostgreSQL
	// 🎯 FIX: Tambahkan postgreDB (*sql.DB) sebagai argumen kedua
	postgresRoutes.RegisterRoutes(app, postgreDB) // Baris 20 sekarang benar

	// 4. Ambil collection MongoDB
	achievementColl := mongodb.GetCollection("uas", "achievements")

	// 5. Route MongoDB Achievement (asumsi ini juga menggunakan koneksi Postgre untuk Auth/RBAC)
	mongoRoutes.RegisterRoutesMongo(app, achievementColl, postgreDB)

	app.Listen(":8080")
}