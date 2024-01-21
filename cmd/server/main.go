package main

import (
	"fmt"
	"net/http"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"
)

func main() {
	var cfg = servconfig.InitConfig()
	var memStor = storage.NewMemoryStorage(cfg)
	var sLogger = logger.NewLogger()

	r := handlers.ChiRouter(memStor)

	sLogger.Info("Server is listening...")
	sLogger.Fatal(http.ListenAndServe(cfg.Address, r))

	/*ctx, cancel := context.WithCancel(context.Background())

	go func() {
		c := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

		<-c
		cancel()
	}()

	httpServer := &http.Server{
		Addr:    cfg.Address,
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
		sLogger.Info("exit reason: %s \n", err)
	}*/

	fmt.Println("Store metrics in file...")
	storage.StoreToFile(memStor, cfg.StoreFile)
}
