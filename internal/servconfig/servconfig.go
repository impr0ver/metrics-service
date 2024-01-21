package servconfig

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"
)

/*
const (
	DefaultListenAddr    = "127.0.0.1:8080"
	DefaultStoreInterval = 300 * time.Second
	DefaultStoreFile     = "/tmp/metrics-db.json"
	RestoreTrue          = true
)

type Config struct {
	ListenAddr    string
	StoreInterval time.Duration
	StoreFile     string
	Restore       bool
}

func SetUpConfig(cfg *Config) {
	var err error

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ListenAddr = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		cfg.StoreInterval, err = time.ParseDuration(v)
		if err != nil {
			cfg.StoreInterval = DefaultStoreInterval
		}
	}
	if v, ok := os.LookupEnv("STORE_FILE"); ok {
		cfg.StoreFile = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		cfg.Restore, err = strconv.ParseBool(v)
		if err != nil {
			cfg.Restore = RestoreTrue
		}
	}
}

func SetUpFlag(cfg *Config) {
	flag.StringVar(&cfg.ListenAddr, "a", DefaultListenAddr, "Server address and port")
	flag.DurationVar(&cfg.StoreInterval, "i", DefaultStoreInterval, "Write store interval")
	flag.StringVar(&cfg.StoreFile, "f", DefaultStoreFile, "Path to store file")
	flag.BoolVar(&cfg.Restore, "r", RestoreTrue, "Restore server metrics flag")
}

func NewConfig() (c Config) {
	SetUpFlag(&c)
	flag.Parse()
	SetUpConfig(&c)
	return
}*/

// /////
type Config struct {
	ListenAddr    string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	Restore       bool          `env:"RESTORE"`
}

func InitConfig() Config {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.ListenAddr, "a", "localhost:8080", "Server address and port.")
	flag.DurationVar(&cfg.StoreInterval, "i", 300*time.Second, "Write store interval")
	flag.StringVar(&cfg.StoreFile, "f", "/tmp/metrics-db.json", "Path to store file")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore server metrics flag")

	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.ListenAddr = envAddr
	}
	if envDuration := os.Getenv("STORE_INTERVAL"); envDuration != "" {
		cfg.StoreInterval, err = time.ParseDuration(envDuration)
		if err != nil {
			cfg.StoreInterval = 300 * time.Second
		}
	}
	if envStorPath := os.Getenv("STORE_FILE"); envStorPath != "" {
		cfg.StoreFile = envStorPath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		cfg.Restore, err = strconv.ParseBool(envRestore)
		if err != nil {
			cfg.Restore = true
		}
	}

	return cfg
}
