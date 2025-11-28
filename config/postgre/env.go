package postgres

import (
	"os"
	"log"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("[POSTGRES] No .env found, using system environment")
	}
}

func GetEnv(key string) string {
	return os.Getenv(key)
}