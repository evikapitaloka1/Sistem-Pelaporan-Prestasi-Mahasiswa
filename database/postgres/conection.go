package db

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
)

var DB *sql.DB

func Connect() {
    dsn := "host=localhost port=5432 user=postgres password=12345678 dbname=uas sslmode=disable"
    var err error
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
