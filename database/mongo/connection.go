package mongodb

import (
    "context"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func Connect() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
        log.Fatal("MongoDB connection failed:", err)
    }

    err = client.Ping(ctx, nil)
    if err != nil {
        log.Fatal("MongoDB ping failed:", err)
    }

    Client = client
    log.Println("MongoDB berhasil connect ke database 'uas'")
}

func GetCollection(dbName, collName string) *mongo.Collection {
    return Client.Database(dbName).Collection(collName)
}
