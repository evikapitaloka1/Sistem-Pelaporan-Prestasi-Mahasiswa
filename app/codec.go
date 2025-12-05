package database

import (
	"context"
	"reflect"
	"time"

	"uas/app/codec"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

func InitMongoDB(uri string) error {
	rb := bsoncodec.NewRegistryBuilder()

	// daftar UUIDCodec
	uuidCodec := &codec.UUIDCodec{}
	rb.RegisterTypeEncoder(reflect.TypeOf(uuid.UUID{}), uuidCodec)
	rb.RegisterTypeDecoder(reflect.TypeOf(uuid.UUID{}), uuidCodec)

	reg := rb.Build()

	client, err := mongo.Connect(context.Background(),
		options.Client().ApplyURI(uri).SetRegistry(reg),
	)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	MongoClient = client
	return nil
}
