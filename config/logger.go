package config

import (
	"os"

	"github.com/gofiber/fiber/v2/middleware/logger"
)

// NewLoggerConfig mengembalikan konfigurasi custom untuk logger Fiber
func NewLoggerConfig() logger.Config {
	return logger.Config{
		// Format log: [Jam] Status - Method Path (Waktu Proses)
		// Contoh: [10:30:00] 200 - GET /api/v1/achievements (12ms)
		Format:     "[${time}] ${status} - ${method} ${path} (${latency})\n",
		TimeFormat: "15:04:05",
		Output:     os.Stdout,
		TimeZone:   "Asia/Jakarta",
	}
}