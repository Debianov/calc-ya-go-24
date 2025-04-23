package main

import "net/http"

func GetDefaultHttpServer(handler http.Handler) *http.Server {
	return &http.Server{Addr: "127.0.0.1:8000", Handler: handler}
}

func GetDefaultGrpcServer() *GrpcTaskServer {
	return &GrpcTaskServer{Addr: "127.0.0.1:5000"}
}
