package main

import (
	pb "github.com/Debianov/calc-ya-go-24/backend/orchestrator/proto"
	"google.golang.org/grpc"
	"net"
)

type GrpcTaskServer struct {
	pb.TaskServiceServer
	Addr string
}

func (g *GrpcTaskServer) ListenAndServe() (err error) {
	var listener net.Listener
	listener, err = net.Listen("tcp", g.Addr)
	if err != nil {
		return
	}
	serviceRegistrar := grpc.NewServer()
	pb.RegisterTaskServiceServer(serviceRegistrar, g)
	err = serviceRegistrar.Serve(listener)
	if err != nil {
		return
	}
	return
}

func StartGrpcServer() (err error) {
	s := GetDefaultGrpcServer()
	err = s.ListenAndServe()
	return
}
