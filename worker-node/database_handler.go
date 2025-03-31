package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./dfs.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create chunks table if it doesn't exist
	createTableSQL := `CREATE TABLE IF NOT EXISTS chunks (
        chunkID TEXT PRIMARY KEY NOT NULL,
        chunkData BLOB NOT NULL
    );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to SQLite and ensured chunks table exists!")
}

func storeChunkInDB(chunkID string, chunkData []byte) error {
	insertChunkSQL := `INSERT INTO chunks (chunkID, chunkData) VALUES (?, ?)`
	_, err := db.Exec(insertChunkSQL, chunkID, chunkData)
	if err != nil {
		return err
	}

	return err
}

func getChunkFromDB(chunkID string) ([]byte, error) {
	getChunkSQL := `SELECT chunkData FROM chunks WHERE chunkID = ?`
	var chunkData []byte

	err := db.QueryRow(getChunkSQL, chunkID).Scan(&chunkData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return chunkData, nil
}

func deleteChunkFromDB(chunkID string) error {
	deleteSQL := `DELETE FROM chunks WHERE chunkID = ?`
	_, err := db.Exec(deleteSQL, chunkID)
	if err != nil {
		return err
	}

	return nil
}
