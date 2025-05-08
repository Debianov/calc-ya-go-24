package main

import (
	"context"
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"github.com/Debianov/calc-ya-go-24/pkg"
	_ "github.com/mattn/go-sqlite3"
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
	db        DbWrapper             = CallDbFabric()
	exprsList CommonExpressionsList = CallEmptyExpressionListFabric()
)

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusNotFound)
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
	dbUser, err = wrapIntoDbUser(jsonUser)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var (
		reqBuf   []byte
		jsonUser *JsonUser
		err      error
	)
	jsonUser = &JsonUser{}
	reqBuf, err = io.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
	}
	err = json.Unmarshal(reqBuf, jsonUser)
	if err != nil {
		log.Panic(err)
	}
	var (
		userFromDb UserWithHashedPassword
	)
	userFromDb, err = db.SelectUser(jsonUser.GetLogin())
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !userFromDb.Is(jsonUser) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var (
		jwtToken        string
		jwtTokenWrapper JwtTokenJsonWrapper
		jwtTokenInBuf   []byte
	)
	jwtToken, err = GenerateJwt(userFromDb)
	if err != nil {
		log.Panic(err)
	}
	jwtTokenWrapper.Token = jwtToken
	jwtTokenInBuf, err = jwtTokenWrapper.Marshal()
	_, err = w.Write(jwtTokenInBuf)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var (
		buf           []byte
		requestStruct RequestJson
		reader        io.ReadCloser
		user          CommonUser
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
	user, err = ParseJwt(requestStruct.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	postfix, ok := pkg.GeneratePostfix(requestStruct.Expression)
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	expr, _ := exprsList.AddExprFabric(user.GetId(), postfix)
	exprIdInJson, err := expr.MarshalId()
	if err != nil {
		log.Panic(err)
	}
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(exprIdInJson)
	if err != nil {
		log.Panic(err)
	}
}

func parseToken(r *http.Request) (user CommonUser, err error) {
	var (
		tokenBuf []byte
		jwtToken JwtTokenJsonWrapper
	)
	tokenBuf, err = io.ReadAll(r.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(tokenBuf, &jwtToken)
	if err != nil {
		return
	}
	user, err = ParseJwt(jwtToken.Token)
	if err != nil {
		return
	}
	return
}

func expressionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var (
		err  error
		user CommonUser
	)
	user, err = parseToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var (
		exprs         []backend.ShortExpression
		exprsFromDb   []backend.ShortExpression
		exprsFromList []backend.CommonExpression
	)
	exprsFromList = exprsList.GetAllOwned(user.GetId())
	for _, expr := range exprsFromList {
		exprs = append(exprs, expr)
	}
	exprsFromDb, err = db.SelectAllExprs(user.GetId())
	if err != nil {
		log.Panic(err)
	}
	exprs = append(exprs, exprsFromDb...)
	slices.SortFunc(exprs, func(expression backend.ShortExpression, expression2 backend.ShortExpression) int {
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
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var (
		err   error
		user  CommonUser
		expr  backend.ShortExpression
		exist bool
	)
	user, err = parseToken(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	idInInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		log.Panic(err)
	}
	expr, err = db.SelectExpr(user.GetId(), int(idInInt))
	if err != nil {
		if expr, exist = exprsList.GetOwned(user.GetId(), int(idInInt)); !exist {
			w.WriteHeader(404)
			return
		}
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
	if expr.GetStatus() == backend.Completed {
		if err = db.InsertExpr(expr); err != nil {
			return nil, status.Errorf(codes.Aborted, "%s", err)
		}
		exprsList.Remove(expr)
	}
	return &pb.Empty{}, status.Error(codes.OK, "")
}
