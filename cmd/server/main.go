package main

import (
	"log"
	"metrics-service/internal/handlers"
	"metrics-service/internal/storage"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	var memStor = storage.MemoryStorage{Gauges: make(map[string]storage.Gauge), Counters: make(map[string]storage.Counter)}

	r.Post("/update/{mType}/{mName}/{mValue}", handlers.MetricsHandlerPost(&memStor))
	r.Get("/value/{mType}/{mName}", handlers.MetricsHandlerGet(&memStor))
	r.Get("/", handlers.MetricsHandlerGetAll(&memStor))

	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(":8080", r))
}
