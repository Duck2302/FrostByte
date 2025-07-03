package main

import (
	"database/sql"
	"fmt"
	"time"
)

// ChunkStorage defines the interface for chunk storage operations
type ChunkStorage interface {
	Store(chunkID string, data []byte) error
	Retrieve(chunkID string) ([]byte, error)
	Delete(chunkID string) error
	Exists(chunkID string) (bool, error)
	Close() error
}

// SQLiteChunkStorage implements ChunkStorage using SQLite
type SQLiteChunkStorage struct {
	db *sql.DB
}

func NewSQLiteChunkStorage(dbPath string) (*SQLiteChunkStorage, error) {
	// Configure SQLite for better concurrency
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_sync=NORMAL&_timeout=10000&_busy_timeout=10000", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Critical: Set connection pool to 1 for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	storage := &SQLiteChunkStorage{db: db}

	// Initialize tables if needed
	if err := storage.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	return storage, nil
}

func (s *SQLiteChunkStorage) initTables() error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS chunks (
        chunkID TEXT PRIMARY KEY NOT NULL,
        chunkData BLOB NOT NULL
    );`
	_, err := s.db.Exec(createTableSQL)
	return err
}

func (s *SQLiteChunkStorage) Store(chunkID string, data []byte) error {
	insertChunkSQL := `INSERT OR REPLACE INTO chunks (chunkID, chunkData) VALUES (?, ?)`
	_, err := s.db.Exec(insertChunkSQL, chunkID, data)
	return err
}

func (s *SQLiteChunkStorage) Retrieve(chunkID string) ([]byte, error) {
	getChunkSQL := `SELECT chunkData FROM chunks WHERE chunkID = ?`
	var chunkData []byte

	err := s.db.QueryRow(getChunkSQL, chunkID).Scan(&chunkData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return chunkData, nil
}

func (s *SQLiteChunkStorage) Delete(chunkID string) error {
	deleteSQL := `DELETE FROM chunks WHERE chunkID = ?`
	_, err := s.db.Exec(deleteSQL, chunkID)
	return err
}

func (s *SQLiteChunkStorage) Exists(chunkID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM chunks WHERE chunkID = ?", chunkID).Scan(&count)
	return count > 0, err
}

func (s *SQLiteChunkStorage) Close() error {
	return s.db.Close()
}
