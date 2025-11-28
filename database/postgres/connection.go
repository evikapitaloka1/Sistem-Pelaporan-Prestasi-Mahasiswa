package postgres

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
    "log"
)

var DB *sql.DB

func Connect() {
    var err error
    DB, err = sql.Open("postgres",
    "host=localhost port=5432 user=postgres password=12345678 dbname=uas sslmode=disable")
    if err != nil {
        log.Fatal("Gagal buka koneksi:", err)
    }

    // cek koneksi
    if err = DB.Ping(); err != nil {
        log.Fatal("Gagal connect ke DB:", err)
    }

    fmt.Println("DB berhasil connect")
}
