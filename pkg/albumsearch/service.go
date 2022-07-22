package album

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type AlbumService interface {
	Find(context.Context, string) ([]Album, error)
	ServiceStatus(context.Context) (int, error)
}

type findAlbumService struct{}

func NewService() AlbumService { return &findAlbumService{} }

type ItunesResponse struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		Artist string `json:"artistName"`
		Title  string `json:"collectionName"`
	} `json:"results"`
}

func (s *findAlbumService) Find(_ context.Context, query string) ([]Album, error) {
	if query == "" {
		return []Album{}, errEmpty
	}

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		logger.Log("Failed to parse user input\n")
	}
	cleanInput := re.ReplaceAllString(query, "")
	cleanInput = strings.ReplaceAll(cleanInput, " ", "+")

	var urlBuilder strings.Builder
	urlBuilder.WriteString(viper.GetString("apiEndpoint"))
	urlBuilder.WriteString("term=")
	urlBuilder.WriteString(cleanInput)
	urlBuilder.WriteString("&entity=album&limit=")
	urlBuilder.WriteString(viper.GetString("resultLimit"))

	url := urlBuilder.String()
	logger.Log("Url: ", url)
	resp, err := http.Get(url)
	if err != nil {
		logger.Log("Failed to fetch results from iTunes Search API\n")
		return []Album{}, errEmpty
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Log("Failed to read from iTunes Search API response\n")
		return []Album{}, errEmpty
	}

	var getResult ItunesResponse
	if err := json.Unmarshal(body, &getResult); err != nil {
		logger.Log("Failed to unmashal iTunes Search API response\n")
		return []Album{}, errEmpty
	}
	var albums []Album

	for _, a := range getResult.Results {
		albums = append(albums, Album{
			Title:  a.Title,
			Artist: a.Artist,
		})
	}
	return albums, nil
}

func (s *findAlbumService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

var errEmpty = errors.New("Query is empty")
