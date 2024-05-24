package servconfig

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/impr0ver/metrics-service/internal/crypt"
)

type Config struct {
	ListenAddr        string          `json:"address"`
	StoreInterval     time.Duration   `json:"store_interval"`
	StoreFile         string          `json:"store_file"`
	Restore           bool            `json:"restore" default:"true"`
	DatabaseDSN       string          `json:"database_dsn"`
	DefaultCtxTimeout time.Duration   `json:"-"`
	Key               string          `json:"-"`
	PathToPrivKey     string          `json:"crypto_key"`
	PrivateKey        *rsa.PrivateKey `json:"-"`
	TrustedSubnet     string          `json:"trusted_subnet"`
}

var (
	defaultListenAddr    = "localhost:8080"
	defaultStoreInterval = 300 * time.Second
	defaultStoreFile     = "/tmp/metrics-db.json"
	defaultRestoreValue  = true
	defaultDSN           = "" //user=postgres password=karat911 host=localhost port=5432 dbname=metrics sslmode=disable
	DefaultCtxTimeout    = 20 * time.Second
	defaultKey           = ""
	defaultPathToPrivKey = ""
	defaultPathToConfig  = ""
	pathToConfig         = defaultPathToConfig
	defaultTrustedSubnet = "192.168.0.0/16"
)

func (c *Config) UnmarshalJSON(data []byte) error {
	type configAlias Config

	customConfig := &struct {
		*configAlias
		StoreInterval string `json:"store_interval"`
	}{
		configAlias: (*configAlias)(c),
	}

	if err := json.Unmarshal(data, customConfig); err != nil {
		return err
	}
	duration, err := time.ParseDuration(customConfig.StoreInterval)
	if err != nil {
		return err
	}
	c.StoreInterval = duration

	return nil
}

func ParseParameters() Config {
	var cfg Config
	var err error

	// first work with config file
	tmpcfg, err := readConfigFile()
	if err == nil {
		if tmpcfg.ListenAddr != "" {
			defaultListenAddr = tmpcfg.ListenAddr
		}
		if tmpcfg.StoreInterval != 0 {
			defaultStoreInterval = tmpcfg.StoreInterval
		}
		if tmpcfg.StoreFile != "" {
			defaultStoreFile = tmpcfg.StoreFile
		}
		if tmpcfg.DatabaseDSN != "" {
			defaultDSN = tmpcfg.DatabaseDSN
		}
		if tmpcfg.Restore != defaultRestoreValue {
			defaultRestoreValue = tmpcfg.Restore
		}
		if tmpcfg.PathToPrivKey != "" {
			defaultPathToPrivKey = tmpcfg.PathToPrivKey
		}
		if tmpcfg.TrustedSubnet != "" {
			defaultTrustedSubnet = tmpcfg.TrustedSubnet
		}
	} else {
		if err.Error() != "no config file" {
			log.Printf("read config error, %v", err)
		}
	}

	// second work with flags
	flag.StringVar(&pathToConfig, "config", defaultPathToConfig, "Path to config")
	flag.StringVar(&cfg.ListenAddr, "a", defaultListenAddr, "Server address and port")
	flag.DurationVar(&cfg.StoreInterval, "i", defaultStoreInterval, "Write store interval")
	flag.StringVar(&cfg.StoreFile, "f", defaultStoreFile, "Path to store file")
	flag.BoolVar(&cfg.Restore, "r", defaultRestoreValue, "Restore server metrics flag")
	flag.StringVar(&cfg.DatabaseDSN, "d", defaultDSN, "Source to DB")
	flag.StringVar(&cfg.Key, "k", defaultKey, "Secret key")
	flag.StringVar(&cfg.PathToPrivKey, "crypto-key", defaultPathToPrivKey, "Private key for asymmetric encoding")
	flag.StringVar(&cfg.TrustedSubnet, "t", defaultTrustedSubnet, "trusted subnet in CIDR format")
	flag.Parse()

	// third work with env's
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ListenAddr = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		cfg.StoreInterval, err = time.ParseDuration(v)
		if err != nil {
			cfg.StoreInterval = defaultStoreInterval
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.StoreFile = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		cfg.Restore, err = strconv.ParseBool(v)
		if err != nil {
			cfg.Restore = defaultRestoreValue
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

	if v, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		cfg.TrustedSubnet = v
	}

	return cfg
}

// readConfigFile - read config file from flag "-config" or env "CONFIG".
func readConfigFile() (Config, error) {
	var pathToConfig string
	tmpcfg := Config{}

	if v, ok := os.LookupEnv("CONFIG"); ok {
		pathToConfig = v
	} else {
		lenArgs := len(os.Args)

		for i, v := range os.Args {
			if (v == "-config") && (i+1 < lenArgs) {
				pathToConfig = os.Args[i+1]
				break
			}
		}
	}
	if pathToConfig == "" {
		return tmpcfg, errors.New("no config file")
	}

	defaultPathToConfig = pathToConfig
	cfgbytes, err := os.ReadFile(pathToConfig)

	if err != nil {
		return tmpcfg, err
	}

	err = json.Unmarshal(cfgbytes, &tmpcfg)
	return tmpcfg, err
}
