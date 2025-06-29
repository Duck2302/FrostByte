package main

import (
	"fmt"
	"log"
	"net/http"
)

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

// validateHTTPMethod checks if the request method matches the expected method
func validateHTTPMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		writeErrorResponse(w, fmt.Sprintf("Only %s requests are allowed", expectedMethod), http.StatusMethodNotAllowed)
		return false
	}
	return true
}
