package main

import (
	db "uas/database/postgres"
	mongodb "uas/database/mongo"

	postgresRoutes "uas/route/postgres"
	mongoRoutes "uas/route/mongo"

	"github.com/gofiber/fiber/v2"
)

func main() {
    db.Connect()
    mongodb.Connect()

    app := fiber.New()

    // Route PostgreSQL
    postgresRoutes.RegisterRoutes(app)

    // Ambil collection MongoDB
    achievementColl := mongodb.GetCollection("uas", "achievements")

    // Route MongoDB Achievement
    mongoRoutes.RegisterRoutesMongo(app, achievementColl, db.GetDB())

    app.Listen(":8080")
}
