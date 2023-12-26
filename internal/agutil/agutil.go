package agutil

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"metrics-service/internal/storage"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
}

func InitConfig(cfg *Config) {
	err := env.Parse(cfg)
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&cfg.Address, "a", "localhost:8080", "Server address and port.")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Frequency of sending metrics to the server.")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Frequency of polling metrics from the package.")
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
}

func SetMetrics(metrics *storage.Metrics, mu *sync.Mutex) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)

	mu.Lock()
	metrics.RuntimeMetrics["Alloc"] = storage.Gauge(rtm.Alloc)
	metrics.RuntimeMetrics["BuckHashSys"] = storage.Gauge(rtm.BuckHashSys)
	metrics.RuntimeMetrics["BuckHashSys"] = storage.Gauge(rtm.BuckHashSys)
	metrics.RuntimeMetrics["Frees"] = storage.Gauge(rtm.Frees)
	metrics.RuntimeMetrics["GCCPUFraction"] = storage.Gauge(rtm.GCCPUFraction)
	metrics.RuntimeMetrics["GCSys"] = storage.Gauge(rtm.HeapAlloc)
	metrics.RuntimeMetrics["HeapAlloc"] = storage.Gauge(rtm.HeapAlloc)
	metrics.RuntimeMetrics["HeapIdle"] = storage.Gauge(rtm.HeapIdle)
	metrics.RuntimeMetrics["HeapInuse"] = storage.Gauge(rtm.HeapInuse)
	metrics.RuntimeMetrics["HeapObjects"] = storage.Gauge(rtm.HeapObjects)
	metrics.RuntimeMetrics["HeapReleased"] = storage.Gauge(rtm.HeapReleased)
	metrics.RuntimeMetrics["HeapSys"] = storage.Gauge(rtm.HeapSys)
	metrics.RuntimeMetrics["LastGC"] = storage.Gauge(rtm.LastGC)
	metrics.RuntimeMetrics["Lookups"] = storage.Gauge(rtm.Lookups)
	metrics.RuntimeMetrics["MCacheInuse"] = storage.Gauge(rtm.MCacheInuse)
	metrics.RuntimeMetrics["MCacheSys"] = storage.Gauge(rtm.MCacheSys)
	metrics.RuntimeMetrics["MSpanInuse"] = storage.Gauge(rtm.MSpanInuse)
	metrics.RuntimeMetrics["MSpanSys"] = storage.Gauge(rtm.MSpanSys)
	metrics.RuntimeMetrics["Mallocs"] = storage.Gauge(rtm.Mallocs)
	metrics.RuntimeMetrics["NextGC"] = storage.Gauge(rtm.NextGC)
	metrics.RuntimeMetrics["NumForcedGC"] = storage.Gauge(rtm.NumForcedGC)
	metrics.RuntimeMetrics["NumGC"] = storage.Gauge(rtm.NumGC)
	metrics.RuntimeMetrics["OtherSys"] = storage.Gauge(rtm.OtherSys)
	metrics.RuntimeMetrics["PauseTotalNs"] = storage.Gauge(rtm.PauseTotalNs)
	metrics.RuntimeMetrics["StackInuse"] = storage.Gauge(rtm.StackInuse)
	metrics.RuntimeMetrics["StackSys"] = storage.Gauge(rtm.StackSys)
	metrics.RuntimeMetrics["Sys"] = storage.Gauge(rtm.Sys)
	metrics.RuntimeMetrics["TotalAlloc"] = storage.Gauge(rtm.TotalAlloc)
	metrics.RuntimeMetrics["RandomValue"] = storage.Gauge(r.Float64())

	metrics.PollCount["PollCount"]++

	mu.Unlock()
}

func InitMetrics(wg *sync.WaitGroup, mu *sync.Mutex, metrics *storage.Metrics, pollInterval int) {
	defer wg.Done()

	for {
		time.Sleep(time.Duration(pollInterval) * time.Second)

		fmt.Println("Set metrics")
		SetMetrics(metrics, mu)
	}
}

func SendMetrics(wg *sync.WaitGroup, mu *sync.Mutex, metrics *storage.Metrics, reportInterval int, URL string) {
	defer wg.Done()

	for {
		time.Sleep(time.Duration(reportInterval) * time.Second)

		mu.Lock()
		metricData := metrics.RuntimeMetrics
		pollCount := metrics.PollCount
		mu.Unlock()

		fmt.Println("Send data")

		for key, value := range metricData {

			fullGaugeURL := fmt.Sprintf("http://%s/update/gauge/%s/%.2f", URL, key, value) //"/update/gauge/someMetric/5.27"
			//fmt.Println(fullGaugeUrl)
			resp, err := http.Post(fullGaugeURL, "text/plain", nil)
			if err != nil {
				fmt.Println(err)
				continue
			}
			resp.Body.Close()
		}
		fullCountURL := fmt.Sprintf("http://%s/update/counter/PollCount/%d", URL, pollCount["PollCount"]) //"/update/counter/PollCount/25"
		//fmt.Println(fullCountUrl)
		resp, err := http.Post(fullCountURL, "text/plain", nil)
		if err != nil {
			fmt.Println(err)
			continue
		}
		resp.Body.Close()
	}
}
