package main

import (
	"flag"
	"metrics-service/internal/agutil"
	"metrics-service/internal/storage"
	"sync"
)

var (
	reportInterval = flag.Int("r", 10, "Frequency of sending metrics to the server.")   //reportInterval time.Duration = 10 * time.Second
	pollInterval   = flag.Int("p", 2, "Frequency of polling metrics from the package.") //pollInterval   time.Duration = 2 * time.Second
	URL            = flag.String("a", "localhost:8080", "Server address and port.")          //URL = "http://127.0.0.1:8080"
)

func main() {
	var metrics = storage.InitMetricsStorage()
	var mu sync.Mutex
	var wg sync.WaitGroup

	flag.Parse()

	wg.Add(2)
	go agutil.InitMetrics(&wg, &mu, &metrics, *pollInterval)
	go agutil.SendMetrics(&wg, &mu, &metrics, *reportInterval, *URL)
	wg.Wait()
}
