package main

import (
	"metrics-service/internal/agutil"
	"metrics-service/internal/storage"
	"sync"
)


func main() {
	var metrics = storage.InitMetricsStorage()
	var mu sync.Mutex
	var wg sync.WaitGroup

	var cfg agutil.Config
	agutil.InitConfig(&cfg)


	wg.Add(2)
	go agutil.InitMetrics(&wg, &mu, &metrics, cfg.PollInterval)
	go agutil.SendMetrics(&wg, &mu, &metrics, cfg.ReportInterval, cfg.Address)
	wg.Wait()
}
