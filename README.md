# Description

The system consists of 3 services, which can be run/deployed independently.

## Core service:
Listens to localhost:8080/search path for a {"query": "SEARCH_TERM"} and forwards the search term to the other services via gRPC

## Album service
Listens to localhost:8082 Calls iTunes Search API with the term received from the core service, returns the response up to 5 items, which can be configured at configs/albumsearch

## Book service
Listens to localhost:8081. Calls Google Book API with the term received from the core service, returns the response up to 5 items, which can be configured at configs/booksearch

# How to run
Start up 3 terminal sessions, one for each service

$ cd .
-   Session 1
$ go run cmd/core/main.go
-   Session 2
$ go run cmd/albumservice/main.go
-   Session 3
$ go run cmd/bookservice/main.go

Send queries via your favorite http client (postman) or via terminal as follows:
curl -X GET \
  -H "Content-type: application/json" \
  -H "Accept: application/json" \
  -d '{"query":"Lord of the rings"}' \
  "http://localhost:8080/search"