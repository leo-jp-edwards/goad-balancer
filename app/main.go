package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	appName := os.Getenv("APP")
	if appName == "" {
		appName = "unknown"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("This is the %s application", appName)
	})

	srv := &http.Server{
		Addr:              ":5000",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Starting %s server on :5000", appName)
	log.Fatal(srv.ListenAndServe())
}
