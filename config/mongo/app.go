package mongo

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InitMongo() *mongo.Database {
	LoadEnv()
	InitLogger()

	uri := GetEnv("MONGO_URI")
	dbName := GetEnv("MONGO_DB")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("[MONGO] Connection error:", err)
	}

	Logger.Println("Connected to MongoDB")

	return client.Database(dbName)
}
