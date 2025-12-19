package config

import (
	"os"

	"github.com/gofiber/fiber/v2/middleware/logger"
)


func NewLoggerConfig() logger.Config {
	return logger.Config{
		
		Format:     "[${time}] ${status} - ${method} ${path} (${latency})\n",
		TimeFormat: "15:04:05",
		Output:     os.Stdout,
		TimeZone:   "Asia/Jakarta",
	}
}