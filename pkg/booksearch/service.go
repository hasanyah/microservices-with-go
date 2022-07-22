package book

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

type BookService interface {
	Find(context.Context, string) ([]Book, error)
	ServiceStatus(context.Context) (int, error)
}

type findBookService struct{}

func NewService() BookService { return &findBookService{} }

type GoogleResponse struct {
	TotalItems int `json:"totalItems"`
	Results    []struct {
		VolumeInfo struct {
			Title   string   `json:"title"`
			Authors []string `json:"authors"`
		} `json:"volumeInfo"`
	} `json:"items"`
}

func (s *findBookService) Find(_ context.Context, query string) ([]Book, error) {
	if query == "" {
		return []Book{}, errEmpty
	}

	re, err := regexp.Compile(`[^\w]`)
	if err != nil {
		logger.Log("Failed to parse user input\n")
	}
	cleanInput := re.ReplaceAllString(query, "")
	cleanInput = strings.ReplaceAll(cleanInput, " ", "+")

	var urlBuilder strings.Builder
	urlBuilder.WriteString(viper.GetString("apiEndpoint"))
	urlBuilder.WriteString("q=")
	urlBuilder.WriteString(cleanInput)
	urlBuilder.WriteString("&maxResults=")
	urlBuilder.WriteString(viper.GetString("resultLimit"))

	url := urlBuilder.String()
	logger.Log("Url: ", url)
	resp, err := http.Get(url)
	if err != nil {
		logger.Log("Failed to fetch results from Google Book Search API\n")
		return []Book{}, errEmpty
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Log("Failed to read from Google Book Search API response\n")
		return []Book{}, errEmpty
	}

	var getResult GoogleResponse
	if err := json.Unmarshal(body, &getResult); err != nil {
		logger.Log("Failed to unmashal Google Book Search API response\n")
		return []Book{}, errEmpty
	}
	var albums []Book

	for _, a := range getResult.Results {
		albums = append(albums, Book{
			Title:  a.VolumeInfo.Title,
			Author: strings.Join(a.VolumeInfo.Authors, ", "),
		})
	}
	return albums, nil
}

func (s *findBookService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}

var errEmpty = errors.New("Query is empty")
