package main

import (
	"database/sql"
	"errors"
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

func getChunkFromDB(chunkID string) ([]byte, error) {
	getChunkSQL := `SELECT chunkData FROM chunks WHERE chunkID =?`
	stmt, err := db.Prepare(getChunkSQL)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(chunkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, errors.New("no rows found")
	}

	var chunkData []byte
	err = rows.Scan(&chunkData)
	if err != nil {
		return nil, err
	}

	return chunkData, nil
}
