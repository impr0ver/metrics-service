package main

import (
	"flag"
	"log"
	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/storage"
	"net/http"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address string `env:"ADDRESS"`
}

func InitConfig(cfg *Config) {
	err := env.Parse(cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Server address and port.")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}
}

func main() {
	var memStor = storage.NewMemoryStorage()
	var cfg Config
	InitConfig(&cfg)

	r := handlers.ChiRouter(memStor)

	log.Println("Server is listening...")
	log.Fatal(http.ListenAndServe(cfg.Address, r))
}
