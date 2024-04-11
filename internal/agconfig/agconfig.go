package agconfig

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/caarlos0/env/v6"
)

type Semaphore struct {
	C chan struct{}
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

type Config struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
}

func InitConfig() Config {

	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Server address and port.")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Frequency of sending metrics to the server.")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Frequency of polling metrics from the package.")
	flag.StringVar(&cfg.Key, "k", "", "Secret key.")
	flag.IntVar(&cfg.RateLimit, "l", 2, "Rate limit.")
	flag.Parse()

	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		cfg.Address = envAddr
	}

	if repInt := os.Getenv("REPORT_INTERVAL"); repInt != "" {
		intVar, err := strconv.Atoi(repInt)
		if err != nil {
			log.Fatal(err)
		}
		cfg.ReportInterval = intVar
	}

	if pollInt := os.Getenv("POLL_INTERVAL"); pollInt != "" {
		intVar, err := strconv.Atoi(pollInt)
		if err != nil {
			log.Fatal(err)
		}
		cfg.PollInterval = intVar
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

	return cfg
}
