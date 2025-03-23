package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var filesCollection *mongo.Collection

func init() {
	// Set client options
	clientOptions := options.Client().ApplyURI("mongodb://mongodb:27017")

	// Connect to MongoDB
	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	// Get a handle for the files collection
	filesCollection = client.Database("dfs").Collection("files")
}

func storeChunkInDB(filename string, chunkID string, workerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update the document for the file, adding the chunk information
	update := bson.M{
		"$push": bson.M{
			fmt.Sprintf("workers.%s", workerID): chunkID,
		},
	}

	_, err := filesCollection.UpdateOne(ctx, bson.M{"filename": filename}, update, options.Update().SetUpsert(true))
	return err
}
