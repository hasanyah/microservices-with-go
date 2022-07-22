package main

import (
	"net"
	"net/http"
	"os"

	"github.com/go-kit/log"

	"microservices-with-go/pkg/core"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func main() {
	logger := log.NewLogfmtLogger(os.Stderr)

	fieldKeys := []string{"method", "error"}
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "assessment_application",
		Subsystem: "user_search_query_propagator",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "assessment_application",
		Subsystem: "user_search_query_propagator",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
	}, fieldKeys)

	var service core.QueryService
	service = core.NewService()
	service = core.LoggingMiddleware{Logger: logger, Next: service}
	service = core.InstrumentingMiddleware{RequestCount: requestCount, RequestLatency: requestLatency, Next: service}

	endpoints := core.NewEndpointSet(service)
	searchQueryHandler := core.NewHTTPHandler(endpoints)

	httpListener, _ := net.Listen("tcp", "localhost:8080")
	http.Serve(httpListener, searchQueryHandler)
}
