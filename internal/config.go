package internal

import "net/http"

func GetDefaultServer() *http.Server {
	return &http.Server{Addr: "127.0.0.1:8000"}
}
