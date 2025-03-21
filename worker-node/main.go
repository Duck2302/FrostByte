package main

import (
	"fmt"
	"net/http"
	"os"
)

func registerWithMaster() {
	hostname, _ := os.Hostname() // Get container name
	masterURL := "http://master:8080/register?id=" + hostname

	resp, err := http.Get(masterURL)
	if err != nil {
		fmt.Println("Failed to register with master:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Registered with master:", hostname)
}

func main() {
	registerWithMaster()

	http.HandleFunc("/worker-task", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Worker %s responding", os.Getenv("HOSTNAME"))
	})

	http.ListenAndServe(":8081", nil)
}
