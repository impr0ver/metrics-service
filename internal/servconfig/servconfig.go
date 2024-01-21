package servconfig

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"FILE_STORAGE_PATH"`
	Restore       bool          `env:"RESTORE"`
}

func InitConfig() *Config {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Server address and port.")
	flag.DurationVar(&cfg.StoreInterval, "i", 300*time.Second, "Write store interval")
	flag.StringVar(&cfg.StoreFile, "f", "/tmp/metrics-db.json", "Path to store file")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore server metrics flag")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}
	if envDuration := os.Getenv("STORE_INTERVAL"); envDuration != "" {
		cfg.StoreInterval, err = time.ParseDuration(envDuration)
		if err != nil {
			cfg.StoreInterval = 300 * time.Second
		}
	}
	if envStorPath := os.Getenv("FILE_STORAGE_PATH"); envStorPath != "" {
		cfg.StoreFile = envStorPath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		cfg.Restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			cfg.Restore = true
		}
	}

	return &cfg
}
