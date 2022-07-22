package book

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
)

type bookSearchRequest struct {
	Query string
}

type Book struct {
	Title  string
	Author string
}

type bookSearchResponse struct {
	Books []Book
	Err   string
}

type serviceStatusRequest struct{}

type serviceStatusResponse struct {
	Status int    `json:"status"`
	Err    string `json:"err,omitempty"`
}

type Set struct {
	SearchEndpoint        endpoint.Endpoint
	ServiceStatusEndpoint endpoint.Endpoint
}

func NewEndpointSet(service BookService) Set {
	return Set{
		SearchEndpoint:        makeBookSearchEndpoint(service),
		ServiceStatusEndpoint: makeServiceStatusEndpoint(service),
	}
}

func (s Set) Find(ctx context.Context, query string) ([]Book, error) {
	resp, err := s.SearchEndpoint(ctx, bookSearchRequest{Query: query})
	if err != nil {
		return []Book{}, errEmpty
	}
	response := resp.(*bookSearchResponse)
	return response.Books, nil
}

func (s Set) ServiceStatus(ctx context.Context) (int, error) {
	resp, err := s.ServiceStatusEndpoint(ctx, serviceStatusRequest{})
	if err != nil {
		return http.StatusNotFound, err
	}
	response := resp.(*serviceStatusResponse)
	return response.Status, nil
}

func makeBookSearchEndpoint(service BookService) endpoint.Endpoint {
	return func(c context.Context, request interface{}) (interface{}, error) {
		req := request.(*bookSearchRequest)
		searchResult, err := service.Find(c, req.Query)
		if err != nil {
			return bookSearchResponse{Books: []Book{}, Err: err.Error()}, nil
		}
		return bookSearchResponse{Books: searchResult, Err: ""}, nil
	}
}

func makeServiceStatusEndpoint(service BookService) endpoint.Endpoint {
	return func(c context.Context, request interface{}) (interface{}, error) {
		req := request.(*serviceStatusRequest)
		fmt.Printf("req: %v\n", req)
		serviceStatus, err := service.ServiceStatus(c)
		if err != nil {
			return serviceStatusResponse{Status: serviceStatus, Err: err.Error()}, nil
		}
		return serviceStatusResponse{Status: serviceStatus, Err: ""}, nil
	}
}
