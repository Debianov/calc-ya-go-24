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

var (
	db, _                                   = CallDbFabric()
	exprsList backend.CommonExpressionsList = backend.CallEmptyExpressionListFabric()
)

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return
	}
	var (
		reqBuf   []byte
		jsonUser = CallJsonUserFabric()
		dbUser   *DbUser
		err      error
	)
	reqBuf, err = io.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(reqBuf, &jsonUser)
	if err != nil {
		log.Panic(err)
	}
	dbUser, err = CallDbUserFabric(jsonUser)
	if err != nil {
		log.Panic(err)
	}
	var (
		lastId int64
	)
	lastId, err = db.InsertUser(dbUser)
	if err != nil {
		log.Panic(err)
	} else {
		dbUser.SetId(lastId)
	}
	return
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return
	}
	var (
		reqBuf   []byte
		jsonUser *JsonUser
		dbUser   *DbUser
		err      error
	)
	reqBuf, err = io.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(reqBuf, jsonUser)
	if err != nil {
		log.Panic(err)
	}
	dbUser, err = CallDbUserFabric(jsonUser)
	if err != nil {
		log.Panic(err)
	}
	var (
		userToCompare DbUser
	)
	userToCompare, err = db.SelectUser(dbUser.GetLogin())
	if err != nil {
		log.Panic(err)
	}
	if dbUser.GetHashedPassword() != userToCompare.GetHashedPassword() {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var (
		jwtToken []byte
	)
	jwtToken, err = backend.GenerateJwt(*dbUser)
	if err != nil {
		log.Panic(err)
	}
	_, err = w.Write(jwtToken)
	if err != nil {
		log.Panic(err)
	}
	return
}

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
	marshaledExpr, err := expr.MarshalId()
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
	slices.SortFunc(exprs, func(expression backend.CommonExpression, expression2 backend.CommonExpression) int {
		if expression.GetId() >= expression2.GetId() {
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
	id := r.PathValue("id")
	idInINt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Panic(err)
	}
	expr, exist := exprsList.Get(int(idInINt))
	if !exist {
		w.WriteHeader(404)
		return
	}
	var exprJsonHandler = backend.ExpressionJsonTitle{Expression: expr}
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
	mux.HandleFunc("/api/v1/register", registerHandler)
	mux.HandleFunc("/api/v1/login", loginHandler)
	mux.HandleFunc("/api/v1/calculate", calcHandler)
	mux.HandleFunc("/api/v1/expressions", expressionsHandler)
	mux.HandleFunc("/api/v1/expressions/{id}", expressionIdHandler)
	handler = panicMiddleware(mux)
	return
}

func (g *GrpcTaskServer) GetTask(_ context.Context, _ *pb.Empty) (result *pb.TaskToSend, err error) {
	expr := exprsList.GetReadyExpr()
	if expr == nil {
		return nil, status.Error(codes.NotFound, "нет готовых задач")
	}
	var taskWithTime backend.GrpcTask
	taskWithTime, err = expr.GetReadyGrpcTask()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%s", err)
	} else {
		result = &pb.TaskToSend{
			PairId:              taskWithTime.GetPairId(),
			Arg1:                taskWithTime.GetArg1(),
			Arg2:                taskWithTime.GetArg2(),
			Operation:           taskWithTime.GetOperation(),
			PermissibleDuration: taskWithTime.GetPermissibleDuration(),
		}
		return result, status.Error(codes.OK, "")
	}
}

func (g *GrpcTaskServer) SendTask(_ context.Context, req *pb.TaskResult) (_ *pb.Empty, err error) {
	timeAtReceiveTask := time.Now()
	exprId, _ := pkg.Unpair(int(req.PairId))
	expr, ok := exprsList.Get(exprId)
	if !ok {
		return nil, status.Error(codes.NotFound, "ID выражения, соответствующей этой задаче, не найдено")
	}
	err = expr.UpdateTask(req, timeAtReceiveTask)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "%s", err)
	}
	return &pb.Empty{}, status.Error(codes.OK, "")
}
