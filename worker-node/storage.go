package main

import (
	"io"
	"os"
	"path/filepath"
)

type ChunkStorage interface {
	Store(chunkID string, data []byte) error
	StoreStream(chunkID string, r io.Reader) error
	Retrieve(chunkID string) ([]byte, error)
	Delete(chunkID string) error
	Exists(chunkID string) (bool, error)
	Close() error
}

type FileChunkStorage struct {
	baseDir string
}

func NewFileChunkStorage(baseDir string) (*FileChunkStorage, error) {
	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		return nil, err
	}
	return &FileChunkStorage{baseDir: baseDir}, nil
}

func (s *FileChunkStorage) chunkPath(chunkID string) string {
	return filepath.Join(s.baseDir, chunkID)
}

func (s *FileChunkStorage) Store(chunkID string, data []byte) error {
	path := s.chunkPath(chunkID)
	return os.WriteFile(path, data, 0644)
}

func (s *FileChunkStorage) StoreStream(chunkID string, r io.Reader) error {
	path := s.chunkPath(chunkID)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *FileChunkStorage) Retrieve(chunkID string) ([]byte, error) {
	path := s.chunkPath(chunkID)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (s *FileChunkStorage) Delete(chunkID string) error {
	path := s.chunkPath(chunkID)
	return os.Remove(path)
}

func (s *FileChunkStorage) Exists(chunkID string) (bool, error) {
	path := s.chunkPath(chunkID)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *FileChunkStorage) Close() error {
	return nil
}
