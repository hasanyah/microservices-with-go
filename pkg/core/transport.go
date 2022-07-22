package core

import (
	"context"
	"encoding/json"
	"net/http"

	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewHTTPHandler(endpoints Set) http.Handler {
	httpHandler := http.NewServeMux()

	httpHandler.Handle("/search", httptransport.NewServer(
		endpoints.SearchEndpoint,
		DecodeSearchRequest,
		EncodeResponse,
	))
	httpHandler.Handle("/status", httptransport.NewServer(
		endpoints.ServiceStatusEndpoint,
		DecodeServiceStatusRequest,
		EncodeResponse,
	))
	httpHandler.Handle("/metrics", promhttp.Handler())
	return httpHandler
}

func DecodeSearchRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request userSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func EncodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	logger.Log("Json encoder: ", response)
	return json.NewEncoder(w).Encode(response)
}

func DecodeServiceStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request serviceStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
