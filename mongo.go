package common

import (
	"context"
	"time"

	"github.com/colin-404/logx"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Database
)

func InitMongoDB(dbName string, mongoURI string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logx.Errorf("Failed to connect to MongoDB: %v", err)
		return err
	}

	// test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		logx.Errorf("Failed to ping MongoDB: %v", err)
		return err
	}

	logx.Infof("Successfully connected to MongoDB")

	mongoClient = client.Database(dbName)

	return nil
}

// InitMongoDB initializes MongoDB connection, if image is true, it will initialize the GridFS bucket
func InitMongoDBWithGridF(dbName string, mongoURI string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		logx.Errorf("Failed to connect to MongoDB: %v", err)
		return err
	}

	// test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		logx.Errorf("Failed to ping MongoDB: %v", err)
		return err
	}

	logx.Infof("Successfully connected to MongoDB")

	mongoClient = client.Database(dbName)

	// Initialize GridFS bucket
	_, err = gridfs.NewBucket(client.Database(dbName), options.GridFSBucket().SetName("images"))
	if err != nil {
		logx.Errorf("Failed to create GridFS bucket: %v", err)
		return err
	}
	logx.Info("Successfully created GridFS bucket for images")

	return nil
}

// CloseMongoDB closes the MongoDB connection
func CloseMongoDB() error {
	if mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return mongoClient.Client().Disconnect(ctx)
	}
	return nil
}

// GetCollection returns a collection from the database
func GetCollection(collectionName string) *mongo.Collection {
	return mongoClient.Collection(collectionName)
}
