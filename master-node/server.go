package main

import (
	"fmt"
	"net/http"
)

type MasterServer struct {
	workerManager  *WorkerManager
	fileOperations *FileOperations
}

func NewMasterServer() *MasterServer {
	wm := NewWorkerManager()
	fo := NewFileOperations(wm)

	return &MasterServer{
		workerManager:  wm,
		fileOperations: fo,
	}
}

func (s *MasterServer) setupRoutes() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
	http.HandleFunc("/register", s.workerManager.registerWorker)
	http.HandleFunc("/workers", s.workerManager.listWorkers)
	http.HandleFunc("/test", s.workerManager.testWorker)
	http.HandleFunc("/upload", s.fileOperations.uploadFile)
	http.HandleFunc("/download/", s.fileOperations.downloadFile)
	http.HandleFunc("/delete", s.fileOperations.deleteFile)
	http.HandleFunc("/files", s.fileOperations.listFiles)
}

func (s *MasterServer) Start(port string) error {
	s.setupRoutes()
	fmt.Printf("Master node listening on :%s\n", port)
	return http.ListenAndServe(":"+port, nil)
}
