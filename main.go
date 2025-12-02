package main

import (
	db "uas/database/postgres"
	mongodb "uas/database/mongo"

	postgresRoutes "uas/route/postgres"
	mongoRoutes "uas/route/mongo"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// 1. Inisialisasi koneksi
	db.Connect()
	mongodb.Connect()

	// 2. Ambil instance koneksi DB PostgreSQL
	postgreDB := db.GetDB() 

	// 3. Ambil instance koneksi MongoDB (*mongo.Client)
	mongoClient := mongodb.GetClient()

	app := fiber.New()

	// 4. Daftarkan Route PostgreSQL (memerlukan client Mongo)
	postgresRoutes.RegisterRoutes(app, postgreDB, mongoClient)

	// 5. Ambil collection MongoDB untuk route Mongo
	achievementColl := mongodb.GetCollection("uas", "achievements")
	mongoRoutes.RegisterRoutesMongo(app, achievementColl, postgreDB)

	app.Listen(":8080")
}
