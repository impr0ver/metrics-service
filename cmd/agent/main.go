package main

import (
	"metrics-service/internal/agutil"
	"metrics-service/internal/storage"
	"sync"
	"time"
)


const (
	reportInterval time.Duration = 10 * time.Second
	pollInterval   time.Duration = 2 * time.Second
	URL                          = "http://127.0.0.1:8080"
)

func main() {
	//var metrics storage.Metrics
	//metrics.RuntimeMetrics = make(map[string]storage.Gauge)
	var metrics = storage.InitMetricsStorage()
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(2)
	go agutil.InitMetrics(&wg, &mu, &metrics, pollInterval)
	go agutil.SendMetrics(&wg, &mu, &metrics, reportInterval, URL)
	wg.Wait()
}
