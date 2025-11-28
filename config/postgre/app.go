package postgres

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func InitPostgres() *sql.DB {
	LoadEnv()
	InitLogger()

	dsn := GetEnv("POSTGRES_DSN")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("[POSTGRES] Connection error:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("[POSTGRES] Ping error:", err)
	}

	Logger.Println("Connected to PostgreSQL")

	return db
}
