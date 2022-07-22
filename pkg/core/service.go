package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	albumtransport "microservices-with-go/pkg/albumsearch"
	booktransport "microservices-with-go/pkg/booksearch"

	"github.com/go-kit/log"
	"google.golang.org/grpc"
)

type mediaObject struct {
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	EntityType string `json:"type"`
}

type QueryService interface {
	Search(context.Context, string) ([]mediaObject, error)
	ServiceStatus(context.Context) (int, error)
}

type userQueryPropagatorService struct{}

func NewService() QueryService { return &userQueryPropagatorService{} }

func (s *userQueryPropagatorService) Search(ctx context.Context, query string) ([]mediaObject, error) {
	if query == "" {
		return []mediaObject{}, errors.New("Query is empty")
	}

	fmt.Fprintf(os.Stdout, "query: %v\n", query)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	bookServiceConnection, err := grpc.Dial("localhost:8081", grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error at bookService: %v\n", err)
	}
	defer bookServiceConnection.Close()

	albumServiceConnection, err := grpc.Dial("localhost:8082", grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dial error at albumService: %v\n", err)
	}
	defer albumServiceConnection.Close()

	bookServiceClient := booktransport.NewGRPCClient(bookServiceConnection)
	bookServiceResult, err := bookServiceClient.Find(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "find endpoint error: %v\n", err)
		bookServiceResult = []booktransport.Book{}
	}

	albumServiceClient := albumtransport.NewGRPCClient(albumServiceConnection)
	albumServiceResult, err := albumServiceClient.Find(ctx, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "find endpoint error: %v\n", err)
		albumServiceResult = []albumtransport.Album{}
	}

	var mediaResult []mediaObject
	for _, b := range bookServiceResult {
		mediaResult = append(mediaResult, mediaObject{
			b.Title,
			b.Author,
			"book",
		})
	}

	for _, b := range albumServiceResult {
		mediaResult = append(mediaResult, mediaObject{
			b.Title,
			b.Artist,
			"album",
		})
	}
	return mediaResult, nil
}

func (s *userQueryPropagatorService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

var logger log.Logger

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
}
