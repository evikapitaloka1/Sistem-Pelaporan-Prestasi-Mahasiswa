package mongodb

import (
	"context"
	"log"
	"os" // Tambahkan os untuk membaca ENV
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client menyimpan koneksi MongoDB global
var Client *mongo.Client

// Connect menginisialisasi koneksi MongoDB
func Connect() {
	// 1. Ambil URI dari Variabel Lingkungan
	mongoURI := os.Getenv("MONGO_URI") 
	if mongoURI == "" {
		log.Fatal("FATAL: MONGO_URI environment variable not set. Please define it in your .env file.")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	// 2. Gunakan URI dari ENV
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI)) 
	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}

	// Ping untuk memastikan koneksi berhasil
	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	log.Println("MongoDB berhasil connect")
}

// GetClient mengembalikan *mongo.Client yang sudah terkoneksi
func GetClient() *mongo.Client {
	return Client
}

// GetCollection mengembalikan *mongo.Collection dari nama database & collection
func GetCollection(dbName, collName string) *mongo.Collection {
	return Client.Database(dbName).Collection(collName)
}