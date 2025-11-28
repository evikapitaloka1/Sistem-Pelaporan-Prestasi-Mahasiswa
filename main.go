package main

import (
	"log"

	db "uas/database/postgres"
	routes "uas/route/postgres"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Connect ke database
	db.Connect()

	// 2. Tampilkan pesan kalau DB berhasil connect
	log.Println("✅ Database berhasil terhubung")

	// 3. Setup Gin router
	r := gin.Default()

	// 4. Register route
	routes.RegisterRoutes(r)

	log.Println("Server running at http://localhost:8080")
	r.Run(":8080")
}
