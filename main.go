package main

import (
	"uas/route/postgres"
	db "uas/database/postgres"
	"github.com/gofiber/fiber/v2"
)

func main() {
	db.Connect() // wajib dipanggil dulu sebelum pake repository/service

	app := fiber.New()
	routes.RegisterRoutes(app)
	app.Listen(":8080")
}
