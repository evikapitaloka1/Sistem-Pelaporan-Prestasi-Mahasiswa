package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Driver PostgreSQL (wajib ada tanda underscore)
)

// Global variable untuk akses database dari repo lain
var PostgresDB *sql.DB

func ConnectPostgres() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	var err error
	// Membuka koneksi dengan driver 'postgres'
	PostgresDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to open connection to PostgreSQL:", err)
	}

	// Cek apakah koneksi benar-benar hidup (Ping)
	err = PostgresDB.Ping()
	if err != nil {
		log.Fatal("Failed to ping PostgreSQL:", err)
	}

	log.Println("✅ Connected to PostgreSQL successfully (Native SQL)")
}