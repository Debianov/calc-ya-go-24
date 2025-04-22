package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/orchestrator/proto"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"google.golang.org/grpc"
	"io"
	"log"
	"net/http"
	"time"
)

type GRPCServer struct {
	pb.TaskServiceServer
}

func (g *GRPCServer) GetTask(_ context.Context, _ *pb.Empty) (result *pb.TaskToSend, err error) {
	expr := exprsList.GetReadyExpr()
	if expr == nil {
		return nil, errors.New("there is no ready task")
	}
	taskToSend := expr.CallTaskToSendFabric()
	if taskToSend.Task == nil {
		return nil, errors.New("BUG: разработчиком ожидается, что выданный expr будеть иметь хотя бы 1" +
			"готовый к отправке task")
	}
	result = &pb.TaskToSend{
		PairId:        int32(taskToSend.Task.PairID),
		Arg1:          taskToSend.Task.Arg1.(int64),
		Arg2:          taskToSend.Task.Arg2.(int64),
		Operation:     taskToSend.Task.Operation,
		OperationTime: taskToSend.Task.OperationTime.String(),
	}
	taskToSend.Task.ChangeStatus(backend.Sent)
	return
}

func (g *GRPCServer) SendTask(ctx context.Context, req *pb.TaskResult) (_ *pb.Empty, err error) {
	var reqInJson = backend.AgentResult{
		ID:     int(req.PairId),
		Result: req.Result,
	}
	exprId, _ := pkg.Unpair(reqInJson.ID)
	expr, ok := exprsList.Get(exprId)
	if !ok {
		return nil, errors.New("ID выражения, соответствующей этой задаче, не найдено")
	}
	err = expr.WriteResultIntoTask(reqInJson.ID, reqInJson.Result, time.Now())
	if err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func NewGRPCServer() *GRPCServer {
	return &GRPCServer{}
}
