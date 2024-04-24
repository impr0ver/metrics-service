package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/impr0ver/metrics-service/internal/agconfig"
	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/agwork"
	"github.com/impr0ver/metrics-service/internal/logger"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// go build -o cmd/agent/agent -ldflags="-X 'main.buildVersion=v9.19' -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')'" cmd/agent/main.go
func buildInfo() {
	fmt.Println("Build version: ", buildVersion)
	fmt.Println("Build date: ", buildDate)
	fmt.Println("Build commit: ", buildCommit)
}

func main() {
	buildInfo()
	var agMemory = agmemory.NewAgMemory()
	var mu sync.RWMutex
	var wg sync.WaitGroup

	cfg := agconfig.InitConfig()

	var sLogger = logger.NewLogger()

	pollIntTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer pollIntTicker.Stop()
	pollIntGopsTicker := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	defer pollIntGopsTicker.Stop()
	repIntTicker := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	defer repIntTicker.Stop()

	donePollInt := make(chan bool)
	doneGopPollInt := make(chan bool)
	doneRepInt := make(chan bool)

	//routine for set runtime metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-donePollInt:
				return
			case t := <-pollIntTicker.C:
				sLogger.Infoln("Set \"runtime\" metrics at", t.Format("04:05"))
				agwork.SetRTMetrics(&agMemory, &mu)
			}
		}
	}()

	//routine for set gops metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneGopPollInt:
				return
			case t := <-pollIntGopsTicker.C:
				sLogger.Infoln("Set \"gops\" metrics at", t.Format("04:05"))
				err := agwork.SetGopsMetrics(&agMemory, &mu)
				if err != nil {
					sLogger.Errorf("error in set gops metrics, %v", err)
				}
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
				agwork.SendMetricsJSONBatch(&mu, &agMemory, cfg.Address, cfg.Key, cfg.RateLimit)
			}
		}
	}()

	wg.Wait()
}
