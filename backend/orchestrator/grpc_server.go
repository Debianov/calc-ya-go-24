package main

import (
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"google.golang.org/grpc"
	"net"
)

type GrpcTaskServer struct {
	pb.TaskServiceServer
	Addr             string
	serviceRegistrar *grpc.Server
}

func (g *GrpcTaskServer) ListenAndServe() (err error) {
	var listener net.Listener
	listener, err = net.Listen("tcp", g.Addr)
	if err != nil {
		return
	}
	g.serviceRegistrar = grpc.NewServer()
	pb.RegisterTaskServiceServer(g.serviceRegistrar, g)
	err = g.serviceRegistrar.Serve(listener)
	if err != nil {
		return
	}
	return
}

func (g *GrpcTaskServer) Close() {
	g.serviceRegistrar.Stop()
}

func UpGrpcServer() (err error) {
	s := GetDefaultGrpcServer()
	err = s.ListenAndServe()
	defer s.Close()
	return
}
