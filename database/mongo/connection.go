package mongo

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client menyimpan koneksi MongoDB global
var Client *mongo.Client

// Connect menginisialisasi koneksi MongoDB
func Connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal("MongoDB connection failed:", err)
	}

	// Ping untuk memastikan koneksi berhasil
	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("MongoDB ping failed:", err)
	}

	log.Println("MongoDB berhasil connect ke database 'uas'")
}

// GetClient mengembalikan *mongo.Client yang sudah terkoneksi
func GetClient() *mongo.Client {
	return Client
}

// GetCollection mengembalikan *mongo.Collection dari nama database & collection
func GetCollection(dbName, collName string) *mongo.Collection {
	return Client.Database(dbName).Collection(collName)
}
