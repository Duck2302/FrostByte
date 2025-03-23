package main

import (
	"log"
	"math/rand"
)

func selectWorker(workers map[string]Worker) string {

	// Create a slice of keys from the map
	keys := make([]string, 0, len(workers))
	for key := range workers {
		keys = append(keys, key)
	}

	// Select a random key from the slice
	randomWorker := keys[rand.Intn(len(keys))]
	if randomWorker == "" {
		log.Fatal("No workers available")
	}
	return randomWorker
}
