package servconfig

import (
	"crypto/rsa"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/impr0ver/metrics-service/internal/crypt"
)

type Config struct {
	ListenAddr        string
	StoreInterval     time.Duration
	StoreFile         string
	Restore           bool
	DatabaseDSN       string
	DefaultCtxTimeout time.Duration
	Key               string
	PathToPrivKey     string
	PrivateKey        *rsa.PrivateKey
}

const (
	DefaultListenAddr    = "localhost:8080"
	DefaultStoreInterval = 300 * time.Second
	DefaultStoreFile     = "/tmp/metrics-db.json"
	RestoreTrue          = true
	DefaultDSN           = "" //user=postgres password=karat911 host=localhost port=5432 dbname=metrics sslmode=disable
	DefaultCtxTimeout    = 20 * time.Second
	DefaultKey           = ""
	DefaultPathToPrivKey = ""
)

func ParseParameters() Config {
	var cfg Config
	var err error

	flag.StringVar(&cfg.ListenAddr, "a", DefaultListenAddr, "Server address and port")
	flag.DurationVar(&cfg.StoreInterval, "i", DefaultStoreInterval, "Write store interval")
	flag.StringVar(&cfg.StoreFile, "f", DefaultStoreFile, "Path to store file")
	flag.BoolVar(&cfg.Restore, "r", RestoreTrue, "Restore server metrics flag")
	flag.StringVar(&cfg.DatabaseDSN, "d", DefaultDSN, "Source to DB")
	flag.StringVar(&cfg.Key, "k", DefaultKey, "Secret key")
	flag.StringVar(&cfg.PathToPrivKey, "crypto-key", DefaultPathToPrivKey, "Private key for asymmetric encoding")
	flag.Parse()

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ListenAddr = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		cfg.StoreInterval, err = time.ParseDuration(v)
		if err != nil {
			cfg.StoreInterval = DefaultStoreInterval
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.StoreFile = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		cfg.Restore, err = strconv.ParseBool(v)
		if err != nil {
			cfg.Restore = RestoreTrue
		}
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = v
	}

	cfg.DefaultCtxTimeout = DefaultCtxTimeout

	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = v
	}

	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.PathToPrivKey = v
	}

	if cfg.PathToPrivKey != "" {
		pKey, err := crypt.InitPrivateKey(cfg.PathToPrivKey)
		if err != nil {
			log.Fatalf("can not init private key, %v", err)
		}
		cfg.PrivateKey = pKey
	}

	return cfg
}
