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
	filesCollection = client.Database("frostbyte").Collection("files")
}

func storeChunkInDB(filename, chunkID, workerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"filename": filename}

	// Store chunks in flat structure since chunkID no longer contains dots
	update := bson.M{
		"$addToSet": bson.M{
			fmt.Sprintf("chunks.%s", chunkID): workerID,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := filesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store chunk in database: %v", err)
	}

	log.Printf("Stored chunk %s for file %s with worker %s", chunkID, filename, workerID)
	return nil
}

// Retrieve file metadata from the database
func GetFileMetadata(ctx context.Context, filename string) (map[string][]string, error) {
	var fileMetadata struct {
		Chunks map[string][]string `bson:"chunks"`
	}
	err := filesCollection.FindOne(ctx, bson.M{"filename": filename}).Decode(&fileMetadata)
	if err != nil {
		return nil, err
	}
	return fileMetadata.Chunks, nil
}

// Delete file metadata from the database
func DeleteFileMetadata(ctx context.Context, filename string) error {
	_, err := filesCollection.DeleteOne(ctx, bson.M{"filename": filename})
	if err != nil {
		return err
	}
	log.Printf("File metadata deleted for file: %s", filename)
	return nil
}

// GetAllFilenames retrieves a list of all filenames in the database
func GetAllFilenames(ctx context.Context) ([]string, error) {
	var filenames []string

	// Find all documents in the collection
	cursor, err := filesCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Iterate through the cursor and extract filenames
	for cursor.Next(ctx) {
		var file struct {
			Filename string `bson:"filename"`
		}
		if err := cursor.Decode(&file); err != nil {
			return nil, err
		}
		filenames = append(filenames, file.Filename)
	}

	// Check for any errors during iteration
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return filenames, nil
}
