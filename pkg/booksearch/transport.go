package book

import (
	"context"
	"fmt"
	book "microservices-with-go/api/book"
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
	book.UnimplementedBookServer
}

func NewGRPCServer(endpoints Set) book.BookServer {
	return &grpcServer{
		find: grpctransport.NewServer(
			endpoints.SearchEndpoint,
			decodeGRPCFindBookRequest,
			encodeGRPCFindBookResponse,
		),
		serviceStatus: grpctransport.NewServer(
			endpoints.ServiceStatusEndpoint,
			decodeGRPCServiceStatusRequest,
			encodeGRPCServiceStatusResponse,
		),
	}
}

func (g *grpcServer) Find(ctx context.Context, r *book.FindBookRequest) (*book.FindBookResponse, error) {
	_, rep, err := g.find.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	logger.Log("Book transport", "Find")
	return rep.(*book.FindBookResponse), nil
}

func (g *grpcServer) ServiceStatus(ctx context.Context, r *book.BookServiceStatusRequest) (*book.BookServiceStatusResponse, error) {
	_, rep, err := g.serviceStatus.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return rep.(*book.BookServiceStatusResponse), nil
}

func (g *grpcServer) mustEmbedUnimplementedBookServer() {

}

func decodeGRPCFindBookRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*book.FindBookRequest)
	logger.Log("Decoding FindBookRequest for: ", req.Query)
	return &bookSearchRequest{Query: req.Query}, nil
}

func decodeGRPCServiceStatusRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*book.BookServiceStatusRequest)
	fmt.Printf("req: %v\n", req)
	return &serviceStatusRequest{}, nil
}

func encodeGRPCFindBookResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(bookSearchResponse)
	logger.Log("Encoding FindBookResponse for: ", reply.Books[0].Title)
	return &book.FindBookResponse{Books: localBookToPbBook(reply.Books), Err: reply.Err}, nil
}

func encodeGRPCServiceStatusResponse(ctx context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(serviceStatusResponse)
	logger.Log("Encoding ServiceStatusResponse for: ", reply.Status)
	return &book.BookServiceStatusResponse{Code: int64(reply.Status), Err: reply.Err}, nil
}

func localBookToPbBook(locals []Book) []*book.Book {
	var pbBooks []*book.Book
	for _, b := range locals {
		logger.Log("Local book conversion to pbBook: ", b.Title)
		pbBooks = append(pbBooks, &book.Book{Title: b.Title, Author: b.Author})
	}
	return pbBooks
}

func NewGRPCClient(conn *grpc.ClientConn) BookService {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))
	var findBookEndpoint endpoint.Endpoint
	{
		findBookEndpoint = grpctransport.NewClient(
			conn,
			"book",
			"Find",
			encodeGRPCFindBookRequest,
			decodeGRPCFindBookResponse,
			book.FindBookResponse{},
		).Endpoint()
		findBookEndpoint = limiter(findBookEndpoint)
		findBookEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Find",
			Timeout: 10 * time.Second,
		}))(findBookEndpoint)
	}

	var bookServiceStatusEndpoint endpoint.Endpoint
	{
		bookServiceStatusEndpoint = grpctransport.NewClient(
			conn,
			"book",
			"ServiceStatus",
			encodeGRPCServiceStatusRequest,
			decodeGRPCServiceStatusResponse,
			book.BookServiceStatusResponse{},
		).Endpoint()
		bookServiceStatusEndpoint = limiter(bookServiceStatusEndpoint)
		bookServiceStatusEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "ServiceStatus",
			Timeout: 10 * time.Second,
		}))(bookServiceStatusEndpoint)
	}

	return Set{
		SearchEndpoint:        findBookEndpoint,
		ServiceStatusEndpoint: bookServiceStatusEndpoint,
	}
}

func encodeGRPCFindBookRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(bookSearchRequest)
	logger.Log("Encoding FindBookRequest for: ", req.Query)
	return &book.FindBookRequest{Query: req.Query}, nil
}

func encodeGRPCServiceStatusRequest(_ context.Context, request interface{}) (interface{}, error) {
	logger.Log("Encoding ServiceStatusRequest for: ", "grpc")
	return &book.BookServiceStatusRequest{}, nil
}

func decodeGRPCFindBookResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	req := grpcRes.(*book.FindBookResponse)
	logger.Log("Decoding FindBookResponse for: ", req.Books[0].Title)
	return &bookSearchResponse{Books: pbBookToLocalBook(req.Books)}, nil
}

func pbBookToLocalBook(pbBooks []*book.Book) []Book {
	var books []Book
	for _, b := range pbBooks {
		logger.Log("PB Book conversion to local: ", b.Title)
		books = append(books, Book{Title: b.Title, Author: b.Author})
	}
	return books
}

func decodeGRPCServiceStatusResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	req := grpcRes.(*book.BookServiceStatusResponse)
	logger.Log("Decoding ServiceStatusResponse for: ", req.Code)
	return &serviceStatusResponse{Status: int(req.Code)}, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
