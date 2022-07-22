package album

import (
	"context"
	"fmt"
	album "microservices-with-go/api/album"
	"os"
	"time"

	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/ratelimit"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/go-kit/log"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type grpcServer struct {
	find          grpctransport.Handler
	serviceStatus grpctransport.Handler
	album.UnimplementedAlbumServer
}

func NewGRPCServer(endpoints Set) album.AlbumServer {
	return &grpcServer{
		find: grpctransport.NewServer(
			endpoints.SearchEndpoint,
			decodeGRPCFindAlbumRequest,
			encodeGRPCFindAlbumResponse,
		),
		serviceStatus: grpctransport.NewServer(
			endpoints.ServiceStatusEndpoint,
			decodeGRPCServiceStatusRequest,
			encodeGRPCServiceStatusResponse,
		),
	}
}

func (g *grpcServer) Find(ctx context.Context, r *album.FindAlbumRequest) (*album.FindAlbumResponse, error) {
	_, rep, err := g.find.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	logger.Log("Album transport", "Find")
	return rep.(*album.FindAlbumResponse), nil
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *album.AlbumServiceStatusRequest) (*album.AlbumServiceStatusResponse, error) {
	_, rep, err := g.serviceStatus.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return rep.(*album.AlbumServiceStatusResponse), nil
}

func (g *grpcServer) mustEmbedUnimplementedAlbumServer() {

}

func decodeGRPCFindAlbumRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*album.FindAlbumRequest)
	logger.Log("Decoding FindAlbumRequest for: ", req.Query)
	return &albumSearchRequest{Query: req.Query}, nil
}

func decodeGRPCServiceStatusRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*album.AlbumServiceStatusRequest)
	fmt.Printf("req: %v\n", req)
	return &serviceStatusRequest{}, nil
}

func encodeGRPCFindAlbumResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(albumSearchResponse)
	logger.Log("Encoding FindAlbumResponse for: ", reply.Albums[0].Title)
	return &album.FindAlbumResponse{Albums: localAlbumToPbAlbum(reply.Albums), Err: reply.Err}, nil
}

func encodeGRPCServiceStatusResponse(ctx context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(serviceStatusResponse)
	logger.Log("Encoding ServiceStatusResponse for: ", reply.Status)
	return &album.AlbumServiceStatusResponse{Code: int64(reply.Status), Err: reply.Err}, nil
}

func localAlbumToPbAlbum(locals []Album) []*album.Album {
	var pbAlbums []*album.Album
	for _, b := range locals {
		logger.Log("Local album conversion to pbAlbum: ", b.Title)
		pbAlbums = append(pbAlbums, &album.Album{Title: b.Title, Artist: b.Artist})
	}
	return pbAlbums
}

func NewGRPCClient(conn *grpc.ClientConn) AlbumService {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))
	var findAlbumEndpoint endpoint.Endpoint
	{
		findAlbumEndpoint = grpctransport.NewClient(
			conn,
			"album",
			"Find",
			encodeGRPCFindAlbumRequest,
			decodeGRPCFindAlbumResponse,
			album.FindAlbumResponse{},
		).Endpoint()
		findAlbumEndpoint = limiter(findAlbumEndpoint)
		findAlbumEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Find",
			Timeout: 10 * time.Second,
		}))(findAlbumEndpoint)
	}

	var albumServiceStatusEndpoint endpoint.Endpoint
	{
		albumServiceStatusEndpoint = grpctransport.NewClient(
			conn,
			"album",
			"ServiceStatus",
			encodeGRPCServiceStatusRequest,
			decodeGRPCServiceStatusResponse,
			album.AlbumServiceStatusResponse{},
		).Endpoint()
		albumServiceStatusEndpoint = limiter(albumServiceStatusEndpoint)
		albumServiceStatusEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "ServiceStatus",
			Timeout: 10 * time.Second,
		}))(albumServiceStatusEndpoint)
	}

	return Set{
		SearchEndpoint:        findAlbumEndpoint,
		ServiceStatusEndpoint: albumServiceStatusEndpoint,
	}
}

func encodeGRPCFindAlbumRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(albumSearchRequest)
	logger.Log("Encoding FindAlbumRequest for: ", req.Query)
	return &album.FindAlbumRequest{Query: req.Query}, nil
}

func encodeGRPCServiceStatusRequest(_ context.Context, request interface{}) (interface{}, error) {
	logger.Log("Encoding ServiceStatusRequest for: ", "grpc")
	return &album.AlbumServiceStatusRequest{}, nil
}

func decodeGRPCFindAlbumResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	req := grpcRes.(*album.FindAlbumResponse)
	logger.Log("Decoding FindAlbumResponse for: ", req.Albums[0].Title)
	return &albumSearchResponse{Albums: pbAlbumToLocalAlbum(req.Albums)}, nil
}

func pbAlbumToLocalAlbum(pbAlbums []*album.Album) []Album {
	var albums []Album
	for _, b := range pbAlbums {
		logger.Log("PB Album conversion to local: ", b.Title)
		albums = append(albums, Album{Title: b.Title, Artist: b.Artist})
	}
	return albums
}

func decodeGRPCServiceStatusResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	req := grpcRes.(*album.AlbumServiceStatusResponse)
	logger.Log("Decoding ServiceStatusResponse for: ", req.Code)
	return &serviceStatusResponse{Status: int(req.Code)}, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
