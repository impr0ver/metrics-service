package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/agwork"
	"github.com/impr0ver/metrics-service/internal/logger"
)

type Config struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	Key            string `env:"KEY"`
}

func InitConfig(cfg *Config) {
	err := env.Parse(cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Server address and port.")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Frequency of sending metrics to the server.")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Frequency of polling metrics from the package.")
	flag.StringVar(&cfg.Key, "k", "", "Secret key.")
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
}

func main() {
	var agMemory = agmemory.NewAgMemory()
	var mu sync.Mutex
	var wg sync.WaitGroup
	var cfg Config
	InitConfig(&cfg)
	var sLogger = logger.NewLogger()

	pollIntTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer pollIntTicker.Stop()
	repIntTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	defer repIntTicker.Stop()

	donePollInt := make(chan bool)
	doneRepInt := make(chan bool)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-donePollInt:
				return
			case t := <-pollIntTicker.C:
				sLogger.Infoln("Set metrics at", t.Format("04:05"))
				agwork.InitMetrics(&mu, &agMemory)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneRepInt:
				return
			case t := <-repIntTicker.C:
				sLogger.Infoln("Send metrics data at", t.Format("04:05"))
				agwork.SendMetricsJSONBatch(&mu, &agMemory, cfg.Address, cfg.Key) //old functions: agwork.SendMetricsJSON and agwork.SendMetrics without JSON
			}
		}
	}()

	wg.Wait()
}
