package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"
	"golang.org/x/sync/errgroup"
)

func main() {
	var cfg = servconfig.InitConfig()
	//cfg := servconfig.NewConfig()
	var memStor = storage.NewMemoryStorage(&cfg)
	var sLogger = logger.NewLogger()

	r := handlers.ChiRouter(memStor)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

		<-c
		cancel()
	}()

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: r,
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		sLogger.Info("Server is listening...")
		return httpServer.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		sLogger.Infof ("Exit reason: %v \n", err)
	}

	if cfg.StoreFile != "" {
		fmt.Println("Store metrics in file...")
		storage.StoreToFile(memStor, cfg.StoreFile)
	}
}
