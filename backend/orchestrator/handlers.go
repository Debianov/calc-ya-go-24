package main

import (
	"context"
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"
)

var exprsList backend.ExpressionsList = backend.CallEmptyExpressionListFabric()

func calcHandler(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)
	if r.Method != http.MethodPost {
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return
	}
	var (
		buf           []byte
		requestStruct backend.RequestJson
		reader        io.ReadCloser
	)
	reader = r.Body
	buf, err = io.ReadAll(reader)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(buf, &requestStruct)
	if err != nil {
		log.Panic(err)
	}
	postfix, ok := pkg.GeneratePostfix(requestStruct.Expression)
	if !ok {
		w.WriteHeader(422)
		return
	}
	expr, _ := exprsList.AddExprFabric(postfix)
	marshaledExpr, err := expr.MarshalID()
	if err != nil {
		log.Panic(err)
	}
	w.WriteHeader(201)
	_, err = w.Write(marshaledExpr)
	if err != nil {
		log.Panic(err)
	}
}

func expressionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	var err error
	exprs := exprsList.GetAllExprs()
	slices.SortFunc(exprs, func(expression *backend.DefaultExpression, expression2 *backend.DefaultExpression) int {
		if expression.ID >= expression2.ID {
			return 0
		} else {
			return -1
		}
	})
	var exprsJsonHandler = backend.ExpressionsJsonTitle{Expressions: exprs}
	exprsHandlerInBytes, err := exprsJsonHandler.Marshal()
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(exprsHandlerInBytes)
	if err != nil {
		log.Panic(err)
	}
}

func expressionIdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}
	var err error
	id := r.PathValue("ID")
	idInINt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Panic(err)
	}
	expr, exist := exprsList.Get(int(idInINt))
	if !exist {
		w.WriteHeader(404)
		return
	}
	var exprJsonHandler = backend.ExpressionJsonTitle{expr}
	exprHandlerInBytes, err := json.Marshal(&exprJsonHandler)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(exprHandlerInBytes)
	if err != nil {
		log.Panic(err)
	}
}

func panicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("response %s, status code: 500", w)
				writeInternalServerError(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(500)
	return
}

func getHandler() (handler http.Handler) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", calcHandler)
	mux.HandleFunc("/api/v1/expressions", expressionsHandler)
	mux.HandleFunc("/api/v1/expressions/{ID}", expressionIdHandler)
	handler = panicMiddleware(mux)
	return
}

func (g *GrpcTaskServer) GetTask(_ context.Context, _ *pb.Empty) (result *pb.TaskToSend, err error) {
	expr := exprsList.GetReadyExpr()
	if expr == nil {
		return nil, status.Error(codes.NotFound, "нет готовых задач")
	}
	taskToSend := expr.GetReadyTask()
	if taskToSend.Task == nil {
		return nil, status.Errorf(codes.Internal, "(bug) разработчиком ожидается, что выданный expr (id %d) "+
			"будет иметь хотя бы 1 готовый к отправке task.", expr.ID)
	}
	result = &pb.TaskToSend{
		PairId:        int32(taskToSend.Task.PairID),
		Arg1:          taskToSend.Task.Arg1.(int64),
		Arg2:          taskToSend.Task.Arg2.(int64),
		Operation:     taskToSend.Task.Operation,
		OperationTime: taskToSend.Task.OperationTime.String(),
	}
	taskToSend.Task.ChangeStatus(backend.Sent)
	return result, status.Error(codes.OK, "")
}

func (g *GrpcTaskServer) SendTask(_ context.Context, req *pb.TaskResult) (_ *pb.Empty, err error) {
	var reqInJson = backend.AgentResult{
		ID:     int(req.PairId),
		Result: req.Result,
	}
	exprId, _ := pkg.Unpair(reqInJson.ID)
	expr, ok := exprsList.Get(exprId)
	if !ok {
		return nil, status.Error(codes.NotFound, "ID выражения, соответствующей этой задаче, не найдено")
	}
	err = expr.UpdateTask(reqInJson.ID, reqInJson.Result, time.Now())
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "%s", err)
	}
	return &pb.Empty{}, status.Error(codes.OK, "")
}
