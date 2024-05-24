package agconfig

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/impr0ver/metrics-service/internal/crypt"
)

type (
	Semaphore struct {
		C chan struct{}
	}

	Config struct {
		Address         string         `env:"ADDRESS" json:"address"`
		PollInterval    time.Duration  `env:"POLL_INTERVAL" json:"poll_interval"`
		ReportInterval  time.Duration  `env:"REPORT_INTERVAL" json:"report_interval"`
		Key             string         `env:"KEY" json:"-"`
		RateLimit       int            `env:"RATE_LIMIT" json:"-"`
		PathToPublicKey string         `env:"CRYPTO_KEY" json:"crypto_key"`
		PublicKey       *rsa.PublicKey `json:"-"`
		RealHostIP      string         `json:"-"`
		GRPCAddress     string         `json:"-"`
	}
)

var (
	DefaultAddress         = "localhost:8080"
	DefaultPollInterval    = 2 * time.Second
	DefaultReportInterval  = 10 * time.Second
	DefaultKey             = ""
	DefaultRateLimit       = 2
	DefaultPathToPublicKey = ""
	DefaultGRPCAddress     = ""
	DefaultPathToConfig    = ""
	pathToConfig           = DefaultPathToConfig
)

func (c *Config) UnmarshalJSON(data []byte) error {
	type configAlias Config

	customConfig := &struct {
		*configAlias
		PollInterval   string `json:"poll_interval"`
		ReportInterval string `json:"report_interval"`
	}{
		configAlias: (*configAlias)(c),
	}

	if err := json.Unmarshal(data, customConfig); err != nil {
		return err
	}
	duration, err := time.ParseDuration(customConfig.ReportInterval)
	if err != nil {
		return err
	}
	c.ReportInterval = duration
	duration, err = time.ParseDuration(customConfig.PollInterval)
	if err != nil {
		return err
	}
	c.PollInterval = duration
	return nil
}

func (s *Semaphore) Acquire() {
	s.C <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.C
}

func NewSemaphore(rateLimit int) *Semaphore {
	return &Semaphore{C: make(chan struct{}, rateLimit)}
}

func InitConfig() Config {

	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	// first work with config file
	tmpcfg, err := readConfigFile()
	if err == nil {
		if tmpcfg.Address != "" {
			DefaultAddress = tmpcfg.Address
		}
		if tmpcfg.ReportInterval != 0 {
			DefaultReportInterval = tmpcfg.ReportInterval
		}
		if tmpcfg.PollInterval != 0 {
			DefaultPollInterval = tmpcfg.PollInterval
		}
		if tmpcfg.PathToPublicKey != "" {
			DefaultPathToPublicKey = tmpcfg.PathToPublicKey
		}
	} else {
		if err.Error() != "no config file" {
			log.Printf("read config error, %v", err)
		}
	}

	// second work with flags
	flag.StringVar(&pathToConfig, "config", DefaultPathToConfig, "path to config")
	flag.StringVar(&cfg.Address, "a", DefaultAddress, "Server address and port.")
	flag.DurationVar(&cfg.ReportInterval, "r", DefaultReportInterval, "Frequency of sending metrics to the server.")
	flag.DurationVar(&cfg.PollInterval, "p", DefaultPollInterval, "Frequency of polling metrics from the package.")
	flag.StringVar(&cfg.Key, "k", DefaultKey, "Secret key.")
	flag.IntVar(&cfg.RateLimit, "l", DefaultRateLimit, "Rate limit.")
	flag.StringVar(&cfg.PathToPublicKey, "crypto-key", DefaultPathToPublicKey, "Public key for asymmetric encoding")
	flag.StringVar(&cfg.GRPCAddress, "rpc", DefaultGRPCAddress, "GRPC server address")
	
	flag.Parse()

	// third work with env's
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}

	if repInt := os.Getenv("REPORT_INTERVAL"); repInt != "" {
		cfg.ReportInterval, err = time.ParseDuration(repInt)
		if err != nil {
			cfg.ReportInterval = DefaultReportInterval
		}
	}

	if pollInt := os.Getenv("POLL_INTERVAL"); pollInt != "" {
		cfg.PollInterval, err = time.ParseDuration(pollInt)
		if err != nil {
			cfg.PollInterval = DefaultPollInterval
		}
	}

	if envKey := os.Getenv("KEY"); envKey != "" {
		cfg.Key = envKey
	}

	if envRLimit := os.Getenv("RATE_LIMIT"); envRLimit != "" {
		intVar, err := strconv.Atoi(envRLimit)
		if err != nil {
			log.Fatal(err)
		}
		cfg.RateLimit = intVar
	}

	if cfg.RateLimit == 0 {
		log.Fatal("rate_limit must not be a zero")
	}

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cfg.PathToPublicKey = envCryptoKey
	}

	if cfg.PathToPublicKey != "" {
		pk, err := crypt.InitPublicKey(cfg.PathToPublicKey)
		if err != nil {
			log.Fatalf("can not init public key, %v", err)
		}
		cfg.PublicKey = pk
	}

	if encGRPCAddr := os.Getenv("GRPC_ADDRESS"); encGRPCAddr != "" {
		cfg.GRPCAddress = encGRPCAddr
	}

	return cfg
}

// readConfigFile - read config file from flag "-config" or env "CONFIG".
func readConfigFile() (Config, error) {
	var pathToConfig string
	tmpcfg := Config{}

	if v, ok := os.LookupEnv("CONFIG"); ok { // CONFIG has priority
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

	DefaultPathToConfig = pathToConfig
	cfgbytes, err := os.ReadFile(pathToConfig)

	if err != nil {
		return tmpcfg, err
	}

	err = json.Unmarshal(cfgbytes, &tmpcfg)
	return tmpcfg, err
}
