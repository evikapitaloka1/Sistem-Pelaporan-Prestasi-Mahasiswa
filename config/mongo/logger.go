package mongo

import (
	"log"
	"os"
)

var Logger *log.Logger

func InitLogger() {
	Logger = log.New(os.Stdout, "[MONGO] ", log.LstdFlags|log.Lshortfile)
}
