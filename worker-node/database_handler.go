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
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        chunkID TEXT NOT NULL,
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
	stmt, err := db.Prepare(insertChunkSQL)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(chunkID, chunkData)
	return err
}
