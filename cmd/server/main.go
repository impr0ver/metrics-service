package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/storage"

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
	var sLogger = logger.NewLogger()

	r := handlers.ChiRouter(memStor, sLogger)

	sLogger.Info("Server is listening...")//log.Println("Server is listening...")
	sLogger.Fatal(http.ListenAndServe(cfg.Address, r))//log.Fatal(http.ListenAndServe(cfg.Address, r))
}
