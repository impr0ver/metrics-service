package main

import (
	"log"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
)

func main() {
	var memStor = storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}
	r := handlers.ChiRouter(&memStor)

	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", r))
}
