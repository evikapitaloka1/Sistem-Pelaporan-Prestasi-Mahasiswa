package db

import (
	"database/sql"
	"log"
	"os" 

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
	// 1. Ambil DSN dari Variabel Lingkungan
	// Pastikan di sini hanya ada KUNCI (nama variabel ENV)
	dsn := os.Getenv("POSTGRES_DSN") 
	if dsn == "" {
		// Log ini yang menyebabkan fatal jika POSTGRES_DSN kosong
		log.Fatal("FATAL: POSTGRES_DSN environment variable not set. Please define it in your .env file.")
	}
	
	var err error
	// 2. Gunakan DSN yang didapat dari ENV
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Postgres connection failed:", err)
	}

	// cek koneksi
	err = DB.Ping()
	if err != nil {
		log.Fatal("Postgres ping failed:", err)
	}

	log.Println("Postgres connected successfully")
}

func GetDB() *sql.DB {
	return DB
}