package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// writeJSONResponse writes a JSON response with the provided data
func writeJSONResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", ContentTypeJSON)
	return json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response with logging
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	log.Printf("Error: %s", message)
	http.Error(w, message, statusCode)
}

// writeSuccessResponse writes a success message response
func writeSuccessResponse(w http.ResponseWriter, message string) {
	fmt.Fprintf(w, "%s", message)
}

// getRequiredParam extracts a required query parameter
func getRequiredParam(r *http.Request, param string) (string, error) {
	value := r.URL.Query().Get(param)
	if value == "" {
		return "", fmt.Errorf("%s parameter is required", param)
	}
	return value, nil
}

func getDownloadPathParameter(r *http.Request) (string, error) {
	fileName := strings.TrimPrefix(r.URL.Path, "/download/")
	fmt.Println("filename searched for %s", fileName)
	if fileName == "" {
		return "", fmt.Errorf("no file was specified")
	}
	return fileName, nil
}

// validateHTTPMethod checks if the request method matches the expected method
func validateHTTPMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		writeErrorResponse(w, fmt.Sprintf("Only %s requests are allowed", expectedMethod), http.StatusMethodNotAllowed)
		return false
	}
	return true
}
