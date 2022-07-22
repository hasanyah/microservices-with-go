package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/log"
	"github.com/spf13/viper"

	bookpb "microservices-with-go/api/book"
	book "microservices-with-go/pkg/booksearch"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/oklog/oklog/pkg/group"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func main() {
	viper.SetConfigName("book")
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
		Subsystem: "book_search_service",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "assessment_application",
		Subsystem: "book_search_service",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	var service book.BookService
	service = book.NewService()
	service = book.LoggingMiddleware{Logger: logger, Next: service}
	service = book.InstrumentingMiddleware{RequestCount: requestCount, RequestLatency: requestLatency, Next: service}

	endpoints := book.NewEndpointSet(service)
	grpcServer := book.NewGRPCServer(endpoints)

	grpcAddress := "localhost:8081"
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
			bookpb.RegisterBookServer(baseServer, grpcServer)
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
