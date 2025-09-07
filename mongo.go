package common

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	defaultURI        = "mongodb://admin:admin@127.0.0.1:27017/?authSource=admin"
	defaultDBName     = "XID"
	defaultClientOnly = false
	defaultTimeout    = 5 * time.Second
)

type Mongo struct {
	mongoClient   *mongo.Client
	mongoDatabase *mongo.Database
}

type MongoOptions struct {
	URI    string
	DBName string
	// if true, will only initialize the client, not the database, default is false
	ClientOnly *bool
}

var defaultMongo atomic.Pointer[Mongo]

func NewMongo(opts *MongoOptions) (*Mongo, error) {
	if opts.URI == "" {
		opts.URI = defaultURI
	}
	if opts.DBName == "" {
		opts.DBName = defaultDBName
	}
	if opts.ClientOnly == nil {
		opts.ClientOnly = &defaultClientOnly
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(opts.URI))
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return nil, err
	}

	err = mongoClient.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf("NEW_MONGO_ERROR %s\n", err.Error())
		return nil, err
	}

	if *opts.ClientOnly {
		defaultMongo.Store(&Mongo{
			mongoClient: mongoClient,
		})
		return defaultMongo.Load(), nil
	}

	mongoDatabase := mongoClient.Database(opts.DBName)
	defaultMongo.Store(&Mongo{
		mongoClient:   mongoClient,
		mongoDatabase: mongoDatabase,
	})

	return defaultMongo.Load(), nil

}

// CloseMongoDB closes the MongoDB connection
func CloseMongoDB() error {
	if defaultMongo.Load().mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return defaultMongo.Load().mongoClient.Disconnect(ctx)
	}
	return nil
}

// GetCollection returns a collection from the database
func GetCollection(collectionName string) *mongo.Collection {
	return defaultMongo.Load().mongoDatabase.Collection(collectionName)
}

func GetMongoCli() *mongo.Client {
	return defaultMongo.Load().mongoClient
}

func GetMongoDatabase() *mongo.Database {
	return defaultMongo.Load().mongoDatabase
}
