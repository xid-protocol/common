package common

import (
	"context"
	"fmt"
	"time"

	"github.com/colin-404/logx"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	MongoClient   *mongo.Client
	MongoDatabase *mongo.Database
	GridFSBucket  *gridfs.Bucket
)

func InitMongoClient(uri string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var opt options.ClientOptions
	opt.SetMaxPoolSize(10)
	opt.SetMinPoolSize(10)

	opt.SetReadPreference(readpref.SecondaryPreferred())
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri), &opt)
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return err
	}

	MongoClient = mongoClient
	return nil
}

func InitMongoDatabase(uri string, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var opt options.ClientOptions
	opt.SetMaxPoolSize(10)
	opt.SetMinPoolSize(10)

	opt.SetReadPreference(readpref.SecondaryPreferred())
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri), &opt)
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return err
	}

	MongoDatabase = mongoClient.Database(dbName)
	return nil
}

// InitMongoDB initializes MongoDB connection, if image is true, it will initialize the GridFS bucket
func InitMongoDBWithGridFS(dbName string, mongoURI string) error {

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

	MongoDatabase = client.Database(dbName)

	// Initialize GridFS bucket
	GridFSBucket, err = gridfs.NewBucket(client.Database(dbName), options.GridFSBucket().SetName("images"))
	if err != nil {
		logx.Errorf("Failed to create GridFS bucket: %v", err)
		return err
	}
	logx.Info("Successfully created GridFS bucket for images")

	return nil
}

// CloseMongoDB closes the MongoDB connection
func CloseMongoDB() error {
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return MongoClient.Disconnect(ctx)
	}
	return nil
}

// GetCollection returns a collection from the database
func GetCollection(collectionName string) *mongo.Collection {
	return MongoDatabase.Collection(collectionName)
}
