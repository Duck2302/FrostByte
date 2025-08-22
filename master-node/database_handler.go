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
	// Set client options with connection pooling
	clientOptions := options.Client().
		ApplyURI("mongodb://mongodb:27017").
		SetMaxPoolSize(50).                   // Maximum number of connections in pool
		SetMinPoolSize(5).                    // Minimum number of connections in pool
		SetMaxConnIdleTime(30 * time.Minute). // Close connections after 30 minutes of inactivity
		SetServerSelectionTimeout(10 * time.Second).
		SetSocketTimeout(30 * time.Second).
		SetConnectTimeout(10 * time.Second)

	// Connect to MongoDB with retry logic
	maxRetries := 10
	retryDelay := 2 * time.Second

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting to connect to MongoDB (attempt %d/%d)...", attempt, maxRetries)

		client, err = mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			log.Printf("Failed to connect to MongoDB (attempt %d): %v", attempt, err)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("Failed to connect to MongoDB after %d attempts: %v", maxRetries, err)
		}

		// Check the connection
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			log.Printf("Failed to ping MongoDB (attempt %d): %v", attempt, err)
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}
			log.Fatalf("Failed to ping MongoDB after %d attempts: %v", maxRetries, err)
		}

		log.Println("Successfully connected to MongoDB!")
		break
	}

	// Get a handle for the files collection
	filesCollection = client.Database("frostbyte").Collection("files")
}

func storeChunkInDB(filename, chunkID, workerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increase timeout
	defer cancel()

	filter := bson.M{"filename": filename}

	// Use array-based storage instead of object keys to avoid field name limitations
	chunkInfo := bson.M{
		"chunkId":  chunkID,
		"workerId": workerID,
	}

	update := bson.M{
		"$addToSet": bson.M{
			"chunks": chunkInfo,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := filesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store chunk in database: %v", err)
	}

	// Verify the update was successful
	if result.ModifiedCount == 0 && result.UpsertedCount == 0 && result.MatchedCount == 0 {
		log.Printf("Warning: No documents were modified when storing chunk %s", chunkID)
	}

	log.Printf("Stored chunk %s for file %s with worker %s", chunkID, filename, workerID)
	return nil
}

func storeFileMetadata(filename string, fileSize int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"filename": filename}
	update := bson.M{
		"$set": bson.M{
			"filename": filename,
			"size":     fileSize,
		},
		"$setOnInsert": bson.M{
			"chunks": []bson.M{},
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := filesCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to store file metadata: %v", err)
	}

	log.Printf("Stored metadata for file %s with size %d", filename, fileSize)
	return nil
}

// Retrieve file metadata from the database
func GetFileMetadata(ctx context.Context, filename string) (map[string][]string, error) {
	var fileMetadata struct {
		Chunks []struct {
			ChunkID  string `bson:"chunkId"`
			WorkerID string `bson:"workerId"`
		} `bson:"chunks"`
	}

	err := filesCollection.FindOne(ctx, bson.M{"filename": filename}).Decode(&fileMetadata)
	if err != nil {
		return nil, err
	}

	// Convert array format back to map format for compatibility
	result := make(map[string][]string)
	for _, chunk := range fileMetadata.Chunks {
		if _, exists := result[chunk.ChunkID]; !exists {
			result[chunk.ChunkID] = []string{}
		}
		result[chunk.ChunkID] = append(result[chunk.ChunkID], chunk.WorkerID)
	}

	return result, nil
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

// FileInfo represents a file with its metadata
type FileInfo struct {
	Filename string `json:"filename" bson:"filename"`
	Size     int64  `json:"size" bson:"size"`
}

// GetAllFilenames retrieves a list of all files with their metadata
func GetAllFilenames(ctx context.Context) ([]FileInfo, error) {
	var files []FileInfo

	// Find all documents in the collection
	cursor, err := filesCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Iterate through the cursor and extract file info
	for cursor.Next(ctx) {
		var file FileInfo
		if err := cursor.Decode(&file); err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	// Check for any errors during iteration
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return files, nil
}
