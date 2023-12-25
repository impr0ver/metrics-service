package main

import (
	"flag"
	"log"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"
)

func main() {
	var memStor = storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}
	addr := flag.String("a", "localhost:8080", "Server address and port.")
	flag.Parse()

	r := handlers.ChiRouter(&memStor)

	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(*addr, r))
}
