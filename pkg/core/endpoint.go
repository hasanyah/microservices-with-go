package core

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

type userSearchRequest struct {
	Query string `json:"query"`
}

type userSearchResponse struct {
	Data []mediaObject `json:"data"`
	Err  string        `json:"err,omitempty"`
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

func NewEndpointSet(service QueryService) Set {
	return Set{
		SearchEndpoint:        makeUserSearchEndpoint(service),
		ServiceStatusEndpoint: makeServiceStatusEndpoint(service),
	}
}

func makeUserSearchEndpoint(service QueryService) endpoint.Endpoint {
	return func(c context.Context, request interface{}) (interface{}, error) {
		req := request.(userSearchRequest)
		searchResult, err := service.Search(c, req.Query)
		if err != nil {
			return userSearchResponse{Data: searchResult, Err: err.Error()}, nil
		}
		return userSearchResponse{Data: searchResult, Err: ""}, nil
	}
}

func makeServiceStatusEndpoint(service QueryService) endpoint.Endpoint {
	return func(c context.Context, request interface{}) (interface{}, error) {
		serviceStatus, err := service.ServiceStatus(c)
		if err != nil {
			return serviceStatusResponse{Status: serviceStatus, Err: err.Error()}, nil
		}
		return serviceStatusResponse{Status: serviceStatus, Err: ""}, nil
	}
}
