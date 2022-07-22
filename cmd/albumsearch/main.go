package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/log"

	albumpb "microservices-with-go/api/album"
	album "microservices-with-go/pkg/albumsearch"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/oklog/oklog/pkg/group"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func main() {
	viper.SetConfigName("album")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../configs/")
	viper.SetDefault("maxNumberResponse", 5)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "assessment_application",
		Subsystem: "album_search_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "assessment_application",
		Subsystem: "album_search_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	var service album.AlbumService
	service = album.NewService()
	service = album.LoggingMiddleware{Logger: logger, Next: service}
	service = album.InstrumentingMiddleware{RequestCount: requestCount, RequestLatency: requestLatency, Next: service}

	endpoints := album.NewEndpointSet(service)
	grpcServer := album.NewGRPCServer(endpoints)

	grpcAddress := "localhost:8082"
	var g group.Group
	{
		grpcListener, err := net.Listen("tcp", grpcAddress)
		if err != nil {
			logger.Log("transport", "gRPC", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "gRPC", "addr", grpcAddress)
			baseServer := grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
			albumpb.RegisterAlbumServer(baseServer, grpcServer)
			return baseServer.Serve(grpcListener)
		}, func(error) {
			grpcListener.Close()
		})
	}
	{
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	logger.Log("exit", g.Run())

}
