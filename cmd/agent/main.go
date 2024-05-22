package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/impr0ver/metrics-service/internal/agconfig"
	"github.com/impr0ver/metrics-service/internal/agmemory"
	"github.com/impr0ver/metrics-service/internal/agwork"
	"github.com/impr0ver/metrics-service/internal/logger"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
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
	var sLogger = logger.NewLogger()

	cfg := agconfig.InitConfig()
	cfg.RealHostIP = agwork.GetHostIP(cfg.Address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	pollIntTicker := time.NewTicker(cfg.PollInterval)
	defer pollIntTicker.Stop()
	pollIntGopsTicker := time.NewTicker(cfg.PollInterval)
	defer pollIntGopsTicker.Stop()
	repIntTicker := time.NewTicker(cfg.ReportInterval)
	defer repIntTicker.Stop()

	//one routine for set runtime metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				stop()
				sLogger.Infoln("- Set \"runtime\" metrics is shutdown...")
				return
			case t := <-pollIntTicker.C:
				sLogger.Infoln("Set \"runtime\" metrics at", t.Format("04:05"))
				agwork.SetRTMetrics(&agMemory, &mu)
			}
		}
	}()

	//one routine for set gops metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				stop()
				sLogger.Infoln("- Set \"gops\" metrics is shutdown...")
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

	//one routine for send metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				stop()
				sLogger.Infoln("- Send metrics is shutdown...")
				return
			case t := <-repIntTicker.C:
				sLogger.Infoln("Send metrics data at", t.Format("04:05"))
				agwork.SendMetricsJSONBatch(&mu, &agMemory, cfg.Address, cfg.Key, cfg.RateLimit, cfg.PublicKey, cfg.RealHostIP)
			}
		}
	}()

	wg.Wait()
	sLogger.Info("Quit signal received, all routines is gracefully shutdown!")

	sLogger.Info("Wait for last metrics send...")
	ctx, cancelFunc := context.WithTimeout(context.Background(), cfg.ReportInterval)
	defer cancelFunc()

	err := lastSendMetrics(ctx, &mu, &agMemory, cfg)
	if err != nil {
		sLogger.Info("lastSendMetrics task exited with error", err)
	}

}

func lastSendMetrics(ctx context.Context, mu *sync.RWMutex, agMemory *agmemory.AgMemory, cfg agconfig.Config) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			agwork.SendMetricsJSONBatch(mu, agMemory, cfg.Address, cfg.Key, cfg.RateLimit, cfg.PublicKey, cfg.RealHostIP)
			time.Sleep(cfg.ReportInterval)
			return nil
		}
	}
}
