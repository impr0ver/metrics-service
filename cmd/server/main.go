package main

import (
	"context"
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
	//cfg := servconfig.NewConfig()
	cfg := servconfig.ParseParameters()
	ctx, cancel := context.WithCancel(context.Background())
	var memStor = storage.NewMemoryStorage(ctx, &cfg)
	var sLogger = logger.NewLogger()

	r := handlers.ChiRouter(memStor)

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
		sLogger.Infof("Exit reason: %v \n", err)
	}

	if cfg.StoreFile != "" {
		sLogger.Info("Store metrics in file...")
		err := storage.StoreToFile(memStor, cfg.StoreFile)
		if err != nil {
			sLogger.Errorf("error to save data in file: %v", err)
		}
	}
}
