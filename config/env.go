package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)


func LoadEnv() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		log.Println(" .env file not found, using system environment variables")
		return
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.Println(" Environment variables loaded")
}

func GetEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}