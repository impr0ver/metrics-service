package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/impr0ver/metrics-service/internal/handlers"
	"github.com/impr0ver/metrics-service/internal/logger"
	"github.com/impr0ver/metrics-service/internal/servconfig"
	"github.com/impr0ver/metrics-service/internal/storage"
	"golang.org/x/sync/errgroup"

	proto "github.com/impr0ver/metrics-service/internal/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	buildInfo()
	var sLogger = logger.NewLogger()
	var rpcSrv *grpc.Server
	cfg := servconfig.ParseParameters()
	ctx, cancel := context.WithCancel(context.Background())
	memStor := storage.NewStorage(ctx, &cfg)

	r := handlers.ChiRouter(memStor, &cfg)

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
		sLogger.Info("gRPC is listening...")
		rpcSrv = StartGRPCServer(cfg, memStor)
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()

		// stop gRPC
		if rpcSrv != nil {
			rpcSrv.GracefulStop()
		}

		// stop REST server
		return httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		sLogger.Infof("Exit reason: %v \n", err)
	}

	// do some work after gracefully shutdown server
	if ok := isNotRunningWithDB(&cfg); ok {
		if cfg.StoreFile != "" {
			sLogger.Info("Store metrics in file...")
			err := storage.StoreToFile(memStor, cfg.StoreFile)
			if err != nil {
				sLogger.Errorf("error to save data in file: %w", err)
			}
		}
	}
}

func isNotRunningWithDB(cfg *servconfig.Config) bool {
	return cfg.DatabaseDSN == ""
}

// go build -o cmd/server/server -ldflags="-X 'main.buildVersion=v9.19' -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')'" cmd/server/main.go
func buildInfo() {
	fmt.Println("Build version: ", buildVersion)
	fmt.Println("Build date: ", buildDate)
	fmt.Println("Build commit: ", buildCommit)
}

func StartGRPCServer(c servconfig.Config, ms storage.MemoryStoragerInterface) *grpc.Server {
	var sLogger = logger.NewLogger()
	// defining the port for the server
	listen, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		sLogger.Info("Can not start listen port 9090")
		return nil
	}

	// creates a gRPC server which has no service registered
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(grpc.UnaryServerInterceptor(handlers.LoggingInterceptor),
		grpc.UnaryServerInterceptor(handlers.VerifyDataInterceptor(c)),
		grpc.UnaryServerInterceptor(handlers.DecryptDataInterceptor(c))))

	// service register
	proto.RegisterMetricsExhangeServer(s, handlers.RPC{Config: c, Ms: ms})
	reflection.Register(s)
	go func() {
		if err := s.Serve(listen); err != nil {
			sLogger.Fatal(err)
		}
	}()
	return s
}