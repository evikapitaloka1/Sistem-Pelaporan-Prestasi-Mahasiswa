
package postgres

import (
	"log"
	"os"
)

var Logger *log.Logger

func InitLogger() {
	Logger = log.New(os.Stdout, "[POSTGRES] ", log.LstdFlags|log.Lshortfile)
}
