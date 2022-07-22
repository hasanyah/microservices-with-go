package album

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
)

type albumSearchRequest struct {
	Query string
}

type Album struct {
	Title  string
	Artist string
}

type albumSearchResponse struct {
	Albums []Album
	Err    string
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

func NewEndpointSet(service AlbumService) Set {
	return Set{
		SearchEndpoint:        makeAlbumSearchEndpoint(service),
		ServiceStatusEndpoint: makeServiceStatusEndpoint(service),
	}
}

func (s Set) Find(ctx context.Context, query string) ([]Album, error) {
	resp, err := s.SearchEndpoint(ctx, albumSearchRequest{Query: query})
	if err != nil {
		return []Album{}, errEmpty
	}
	response := resp.(*albumSearchResponse)
	return response.Albums, nil
}

func (s Set) ServiceStatus(ctx context.Context) (int, error) {
	resp, err := s.ServiceStatusEndpoint(ctx, serviceStatusRequest{})
	if err != nil {
		return http.StatusNotFound, err
	}
	response := resp.(*serviceStatusResponse)
	return response.Status, nil
}

func makeAlbumSearchEndpoint(service AlbumService) endpoint.Endpoint {
	return func(c context.Context, request interface{}) (interface{}, error) {
		req := request.(*albumSearchRequest)
		searchResult, err := service.Find(c, req.Query)
		if err != nil {
			return albumSearchResponse{Albums: []Album{}, Err: err.Error()}, nil
		}
		return albumSearchResponse{Albums: searchResult, Err: ""}, nil
	}
}

func makeServiceStatusEndpoint(service AlbumService) endpoint.Endpoint {
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
