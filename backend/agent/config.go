package main

import (
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getDefaultAgent(conn *grpc.ClientConn) pb.TaskServiceClient {
	return pb.NewTaskServiceClient(conn)
}

func getDefaultGrpcClient() (conn *grpc.ClientConn, err error) {
	return grpc.NewClient("127.0.0.1:5000", grpc.WithTransportCredentials(insecure.NewCredentials()))
}
