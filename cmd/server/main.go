package main

import (
	"log"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	var memStor = storage.InitMemory()

	mux.HandleFunc("/update/", handlers.MetricsHandler(&memStor))
	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
