package mongo

import (
	"context"
	"log"
	"reflect"
	"time"
	"uas/app/codec"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

// Connect — koneksi mongodb + registry custom UUID Codec
func Connect() {
	// 1. Registry untuk UUID Codec
	rb := bson.NewRegistryBuilder()

	uuidCodec := &codec.UUIDCodec{}
	rb.RegisterTypeEncoder(reflect.TypeOf(uuid.UUID{}), uuidCodec)
	rb.RegisterTypeDecoder(reflect.TypeOf(uuid.UUID{}), uuidCodec)

	registry := rb.Build()

	// 2. Setup connection
	clientOptions := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetRegistry(registry)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Mongo connection error: %v", err)
	}

	// Test ping
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Mongo ping failed: %v", err)
	}

	log.Println("MongoDB connected with UUID codec.")
}

// GetClient — ambil instance *mongo.Client
func GetClient() *mongo.Client {
	return client
}

// GetCollection — ambil collection
func GetCollection(dbName, collName string) *mongo.Collection {
	return client.Database(dbName).Collection(collName)
}
