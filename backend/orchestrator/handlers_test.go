package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var compareTemplate = "ожидается \"%s\", получен \"%s\""

/*
testThroughHttpHandler запускает все тесты через handler, используя параметры casesHandler.
Генерируемый запрос всегда отправляется с заголовком "Content-Type": "application/json".
*/
func testThroughHttpHandler[K, V backend.JsonPayload](handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	casesHandler backend.HttpCasesHandler[K, V], compareFunc func(t *testing.T, w *httptest.ResponseRecorder,
		casesHandler backend.CasesHandler, currentTestCase backend.ByteCase)) {
	var (
		cases []backend.ByteCase
		err   error
	)
	cases, err = backend.ConvertToByteCases(casesHandler.RequestsToSend, casesHandler.ExpectedResponses)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		var (
			w      = httptest.NewRecorder()
			reader = bytes.NewReader(testCase.ToSend)
			req    = httptest.NewRequest(casesHandler.HttpMethod, casesHandler.UrlTarget, reader)
		)
		req.Header.Set("Content-Type", "application/json")
		handler(w, req)
		compareFunc(t, w, &casesHandler, testCase)
	}
}

/*
testThroughServeMux работает также, как и testThroughHttpHandler, но с обёрткой handler-а в http.ServerMux.
Необходимо для тестирования некоторых handler-ов, которые вызывают методы, связанные с парсингом URL в запросах
к handler-у (например, request.PathValue): парсинг происходит только при вызове http.ServerMux
(https://pkg.go.dev/net/http#ServeMux).
*/
func testThroughServeMux[K, V backend.JsonPayload](
	handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	casesHandler backend.ServerMuxHttpCasesHandler[K, V], compareFunc func(t *testing.T, w *httptest.ResponseRecorder,
		casesHandler backend.CasesHandler, currentTestCase backend.ByteCase)) {
	var (
		cases []backend.ByteCase
		err   error
	)
	cases, err = backend.ConvertToByteCases(casesHandler.RequestsToSend, casesHandler.ExpectedResponses)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		var (
			w         = httptest.NewRecorder()
			reader    = bytes.NewReader(testCase.ToSend)
			req       = httptest.NewRequest(casesHandler.HttpMethod, casesHandler.UrlTarget, reader)
			serverMux = http.NewServeMux()
		)
		req.Header.Set("Content-Type", "application/json")
		serverMux.HandleFunc(casesHandler.UrlTemplate, handler)
		serverMux.ServeHTTP(w, req)
		compareFunc(t, w, &casesHandler, testCase)
	}
}

func defaultCmpFunc(t *testing.T, w *httptest.ResponseRecorder,
	casesHandler backend.CasesHandler, currentTestCase backend.ByteCase) {
	if casesHandler.GetExpectedHttpCode() != w.Code {
		t.Errorf(compareTemplate, strconv.Itoa(casesHandler.GetExpectedHttpCode()),
			strconv.Itoa(w.Code))
	}
	if bytes.Compare(currentTestCase.Expected, w.Body.Bytes()) != 0 {
		t.Errorf(compareTemplate, currentTestCase.Expected, w.Body.Bytes())
	}
}

/*
testByStructCompareThroughHttpHandler переводит case-ы в backend.ByteCase, но
сравнивает структуры, что может быть принципиально важно, если идёт тестирование
handler-ов, которые возвращают байт-строки с данными, нуждающиеся в
манипуляциях для того, чтобы они были пригодны для сравнения
Например, jwt-токены. Их нельзя сравнивать напрямую, т.к в любой отрезок времени
возвращается уникальная последовательность символов.
*/
func testByStructCompareThroughHttpHandler[K backend.JsonPayload, V parsedToken](
	handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	casesHandler backend.HttpCasesHandler[K, V]) {
	var (
		cases      []backend.ByteCase
		emptyJsons = make([]backend.EmptyJson, len(casesHandler.ExpectedResponses)) // ExpectedResponses не будут
		// использоваться, поэтому мы заменяем их на пустые json-ы. Однако, мы должны передавать такое количество,
		//которое будет = casesHandler.RequestsToSend
		err error
	)
	cases, err = backend.ConvertToByteCases(casesHandler.RequestsToSend, emptyJsons)
	if err != nil {
		t.Fatal(err)
	}
	for ind, testCase := range cases {
		var (
			w      = httptest.NewRecorder()
			reader = bytes.NewReader(testCase.ToSend)
			req    = httptest.NewRequest(casesHandler.HttpMethod, casesHandler.UrlTarget, reader)
		)
		req.Header.Set("Content-Type", "application/json")
		handler(w, req)
		if casesHandler.GetExpectedHttpCode() != w.Code {
			t.Errorf(compareTemplate, strconv.Itoa(casesHandler.GetExpectedHttpCode()),
				strconv.Itoa(w.Code))
		}
		var (
			err       error
			respBuf   []byte
			realToken JwtTokenJsonWrapper
		)
		respBuf, err = io.ReadAll(w.Body)
		if err != nil {
			t.Fatal(err)
		}
		err = json.Unmarshal(respBuf, &realToken)
		if err != nil {
			t.Fatal(err)
		}
		var (
			userFromParsedToken   CommonUser
			userFromExpectedToken = casesHandler.ExpectedResponses[ind].GetExpectedUser()
		)
		userFromParsedToken, err = ParseJwt(realToken.Token)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, userFromExpectedToken.GetLogin(), userFromParsedToken.GetLogin())
		assert.Equal(t, userFromExpectedToken.GetId(), userFromParsedToken.GetId())
	}
}

var testUser = UserStub{
	Login:    "test",
	Password: "qwerty",
	id:       0,
}
var token, _ = GenerateJwt(&testUser)

func TestCalcHandler(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	db = callStubDbWithRegisteredUserFabric(testUser)

	t.Run("201Code", func(t *testing.T) {
		var (
			requestsToTest = []*backend.RequestJsonStub{{Token: token, Expression: "2+2*4"},
				{Token: token, Expression: "4*2+3*5"}}
			expectedResponses = []*backend.ExpressionJsonStub{{ID: 0}, {ID: 1}}
			commonHttpCase    = backend.HttpCasesHandler[*backend.RequestJsonStub, *backend.ExpressionJsonStub]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "POST",
				UrlTarget: "/api/v1/calculate", ExpectedHttpCode: http.StatusCreated}
		)
		testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("422Code", func(t *testing.T) {
		var (
			requestsToTest = []*backend.RequestJsonStub{{Token: token, Expression: "2++2*4"},
				{Token: token, Expression: "4*(2+3"}, {Token: token, Expression: "8+2/3)"},
				{Token: token, Expression: "4*()2+3"}}
			expectedResponses = []backend.EmptyJson{{}, {}, {}, {}}
			commonHttpCase    = backend.HttpCasesHandler[*backend.RequestJsonStub, backend.EmptyJson]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
				ExpectedHttpCode: http.StatusUnprocessableEntity}
		)
		testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("404Code", func(t *testing.T) {
		var (
			requestsToTest    = []*backend.RequestJsonStub{{Token: token, Expression: "2+2*4"}}
			expectedResponses = []*backend.EmptyJson{{}}
			commonHttpCase    = backend.HttpCasesHandler[*backend.RequestJsonStub, *backend.EmptyJson]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/calculate",
				ExpectedHttpCode: http.StatusNotFound}
		)
		testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)
	})
}

func testExpressionsHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	db = callStubDbWithRegisteredUserFabric(testUser)

	t.Run("FromDbAndExprsList", func(t *testing.T) {
		t.Cleanup(func() {
			exprsList = CallEmptyExpressionListFabric()
			db.(*DbStub).FlushExprs()
		})
		var (
			expectedExpressionsFromList = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0},
				{Id: 2, Status: backend.Cancelled, Result: 0}, {Id: 3, Status: backend.NoReadyTasks, Result: 0}}
			expectedExpressionsFromDb = []backend.ExpressionStub{{Id: 1, Status: backend.Completed, Result: 432},
				{Id: 4, Status: backend.Completed, Result: -2345}}
			expectedExpressions []backend.ExpressionStub
		)
		db.(*DbStub).InsertExprs(testUser.GetId(), expectedExpressionsFromDb)
		exprsList = callExprsListStubFabric(testUser.GetId(), expectedExpressionsFromList...)
		expectedExpressions = append(expectedExpressions, expectedExpressionsFromList...)
		expectedExpressions = append(expectedExpressions, expectedExpressionsFromDb...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionsJsonTitleStub{{expectedExpressions}}
			commonHttpCase    = backend.HttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.ExpressionsJsonTitleStub]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTarget: "/api/v1/expressions", ExpectedHttpCode: http.StatusOK}
		)
		testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("FromExprsList", func(t *testing.T) {
		var (
			expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0},
				{Id: 1, Status: backend.Completed, Result: 432}, {Id: 2, Status: backend.Cancelled, Result: 0},
				{Id: 3, Status: backend.NoReadyTasks, Result: 0}, {Id: 4, Status: backend.Completed, Result: -2345}}
		)
		exprsList = callExprsListStubFabric(testUser.GetId(), expectedExpressions...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionsJsonTitleStub{{expectedExpressions}}
			commonHttpCase    = backend.HttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.ExpressionsJsonTitleStub]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTarget: "/api/v1/expressions", ExpectedHttpCode: http.StatusOK}
		)
		testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("FromDb", func(t *testing.T) {
		t.Cleanup(func() {
			db.(*DbStub).FlushExprs()
		})
		var (
			expectedExpressions = []backend.ExpressionStub{{Id: 1, Status: backend.Completed, Result: 432},
				{Id: 4, Status: backend.Completed, Result: -2345}}
		)
		db.(*DbStub).InsertExprs(testUser.GetId(), expectedExpressions)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionsJsonTitleStub{{expectedExpressions}}
			commonHttpCase    = backend.HttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.ExpressionsJsonTitleStub]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTarget: "/api/v1/expressions", ExpectedHttpCode: http.StatusOK}
		)
		testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("EmptyStorages", func(t *testing.T) {
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionsJsonTitle{{Expressions: make([]backend.ShortExpression, 0)}}
			commonHttpCase    = backend.HttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.ExpressionsJsonTitle]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTarget: "/api/v1/expressions", ExpectedHttpCode: http.StatusOK}
		)
		testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
	})
}

func testExpressionsHandler404(t *testing.T) {
	t.Run("WrongMethod", func(t *testing.T) {
		t.Cleanup(func() {
			exprsList = CallEmptyExpressionListFabric()
		})
		var (
			expressionToList = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
		)
		exprsList = callExprsListStubFabric(testUser.GetId(), expressionToList...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.EmptyJson{{}}
			commonHttpCase    = backend.HttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.EmptyJson]{RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/expressions",
				ExpectedHttpCode: http.StatusNotFound}
		)
		testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
	})
}

func TestExpressionHandler(t *testing.T) {
	exprsList = CallEmptyExpressionListFabric()
	t.Run("200Code", testExpressionsHandler200)
	t.Run("404Code", testExpressionsHandler404)
}

func testExpressionIdHandler200(t *testing.T) {
	t.Run("FromDb", func(t *testing.T) {
		t.Cleanup(func() {
			db.(*DbStub).FlushExprs()
		})
		var (
			expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 12}}
		)
		db.(*DbStub).InsertExprs(testUser.GetId(), expectedExpressions)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionJsonTitleStub{{expectedExpressions[0]}}
			serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[*JwtTokenJsonWrapperStub,
				*backend.ExpressionJsonTitleStub]{RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses,
				HttpMethod: "GET", UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
				ExpectedHttpCode: http.StatusOK}
		)
		testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
	})
	t.Run("FromExprsList", func(t *testing.T) {
		t.Cleanup(func() {
			exprsList = CallEmptyExpressionListFabric()
		})
		var (
			expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 23}}
		)
		exprsList = callExprsListStubFabric(testUser.GetId(), expectedExpressions...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.ExpressionJsonTitleStub{{expectedExpressions[0]}}
			serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[*JwtTokenJsonWrapperStub,
				*backend.ExpressionJsonTitleStub]{RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses,
				HttpMethod: "GET", UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
				ExpectedHttpCode: http.StatusOK}
		)
		testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
	})
}

func testExpressionIdHandler404(t *testing.T) {
	t.Run("WrongId", func(t *testing.T) {
		t.Cleanup(func() {
			exprsList = CallEmptyExpressionListFabric()
		})
		var (
			expressionToList = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
		)
		exprsList = callExprsListStubFabric(testUser.GetId(), expressionToList...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.EmptyJson{{}}
			serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.EmptyJson]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/1",
				ExpectedHttpCode: http.StatusNotFound}
		)
		testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
	})
	t.Run("WrongMethod", func(t *testing.T) {
		var (
			expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
		)
		exprsList = callExprsListStubFabric(testUser.GetId(), expectedExpressions...)
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.EmptyJson{{}}
			serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.EmptyJson]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "POST",
				UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
				ExpectedHttpCode: http.StatusNotFound}
		)
		testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
	})
	t.Run("EmptyStorages", func(t *testing.T) {
		var (
			requestsToTest    = []*JwtTokenJsonWrapperStub{{Token: token}}
			expectedResponses = []*backend.EmptyJson{{}}
			serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[*JwtTokenJsonWrapperStub, *backend.EmptyJson]{
				RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
				UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
				ExpectedHttpCode: http.StatusNotFound}
		)
		testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
	})
}

func TestExpressionIdHandler(t *testing.T) {
	exprsList = CallEmptyExpressionListFabric()
	t.Run("200Code", testExpressionIdHandler200)
	t.Run("404Code", testExpressionIdHandler404)
}

func TestPanicMiddlewareGood(t *testing.T) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", stubHandlerWithoutPanic)
	var (
		middlewareHandler = panicMiddleware(mux)
		w                 = httptest.NewRecorder()
		mockReader        = bytes.NewReader(nil)
		req               = httptest.NewRequest("POST", "/api/v1/calculate", mockReader)
	)
	middlewareHandler.ServeHTTP(w, req)
	if 200 != w.Code {
		t.Errorf(compareTemplate, "200", strconv.Itoa(w.Code))
	}
}

func stubHandlerWithoutPanic(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	return
}

func TestPanicMiddlewareBad(t *testing.T) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", mockHandlerWithPanic)
	middlewareHandler := panicMiddleware(mux)
	var (
		w          = httptest.NewRecorder()
		mockReader = bytes.NewReader(nil)
		req        = httptest.NewRequest("GET", "/api/v1/calculate", mockReader)
	)
	middlewareHandler.ServeHTTP(w, req)
	if 500 != w.Code {
		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
	}
}

func mockHandlerWithPanic(_ http.ResponseWriter, _ *http.Request) {
	panic(errors.New("ААААААА!!!!"))
}

func TestInternalServerErrorHandler(t *testing.T) {
	var (
		w = httptest.NewRecorder()
	)
	writeInternalServerError(w)
	if 500 != w.Code {
		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
	}
}

func testGetTaskNotFoundCode(t *testing.T) {
	var (
		g      = GetDefaultGrpcServer()
		result *pb.TaskToSend
		err    error
	)
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	exprsList = callExprsEmptyListFabric()
	result, err = g.GetTask(context.TODO(), &pb.Empty{})
	assert.Equal(t, codes.NotFound, status.Code(err))
	nilToTaskToSend := (*pb.TaskToSend)(nil) // возвращается не просто nil
	assert.Equal(t, nilToTaskToSend, result)
}

func testGetTaskInternalCode(t *testing.T) {
	var (
		g      = GetDefaultGrpcServer()
		result *pb.TaskToSend
		err    error
	)
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	exprsList = callExprsListStubFabric(testUser.GetId(), backend.ExpressionStub{
		Id:           0,
		Status:       backend.Ready,
		TasksHandler: &backend.TasksHandlerStub{},
	})
	result, err = g.GetTask(context.TODO(), &pb.Empty{})
	assert.Equal(t, codes.Internal, status.Code(err))
	nilToTaskToSend := (*pb.TaskToSend)(nil) // возвращается не просто nil
	assert.Equal(t, nilToTaskToSend, result)
}

func testGetTaskOkCode(t *testing.T) {
	var (
		g      = GetDefaultGrpcServer()
		result *pb.TaskToSend
		err    error
	)
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedTask = backend.CallTaskFabric(0, 2, 4, "+", backend.ReadyToCalc)
	)
	exprsList = callExprsListStubFabric(testUser.GetId(), backend.ExpressionStub{
		Id:           0,
		Status:       backend.Ready,
		TasksHandler: &backend.TasksHandlerStub{Buf: map[int32]backend.InternalTask{0: expectedTask}}})
	result, err = g.GetTask(context.TODO(), &pb.Empty{})
	assert.Equal(t, codes.OK, status.Code(err))
	arg1, _ := expectedTask.GetArg1()
	arg2, _ := expectedTask.GetArg2()
	var (
		wrappedExpectedTask = &pb.TaskToSend{
			PairId:              expectedTask.GetPairId(),
			Arg1:                arg1,
			Arg2:                arg2,
			Operation:           expectedTask.GetOperation(),
			PermissibleDuration: expectedTask.GetPermissibleDuration().String(),
		}
	)
	assert.EqualExportedValues(t, wrappedExpectedTask, result)
}

func TestGetTask(t *testing.T) {
	t.Run("NotFoundCode", testGetTaskNotFoundCode)
	t.Run("InternalCode", testGetTaskInternalCode)
	t.Run("OkCode", testGetTaskOkCode)
}

func testSendTaskNotFoundCode(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		g      = GetDefaultGrpcServer()
		toSend = &pb.TaskResult{
			PairId: 12,
			Result: 0,
		}
		err error
	)
	exprsList = callExprsEmptyListFabric()
	_, err = g.SendTask(context.TODO(), toSend)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func testSendTaskAbortedCode(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		g      = GetDefaultGrpcServer()
		toSend = &pb.TaskResult{
			PairId: 0,
			Result: 2,
		}
		err          error
		tasksHandler = &backend.TasksHandlerStub{Buf: make(map[int32]backend.InternalTask)}
	)
	exprsList = callExprsListStubFabric(testUser.GetId(), backend.ExpressionStub{
		Id:           0, // pairId 15 = expr id 3 and internal task Id (unused) 3
		Status:       backend.Ready,
		Result:       0,
		TasksHandler: tasksHandler,
	})
	_, err = g.SendTask(context.TODO(), toSend)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func testSendTaskOkCode(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		g              = GetDefaultGrpcServer()
		expectedResult = int64(15)
		expectedPairId = int32(0)
		toSend         = &pb.TaskResult{
			PairId: expectedPairId,
			Result: expectedResult,
		}
		err          error
		tasksHandler = &backend.TasksHandlerStub{Buf: map[int32]backend.InternalTask{expectedPairId: backend.CallTaskFabric(
			expectedPairId, 2, 3, "-", backend.ReadyToCalc)}}
	)
	exprsList = callExprsListStubFabric(testUser.GetId(), backend.ExpressionStub{
		Id:           0,
		Status:       backend.Ready,
		Result:       0,
		TasksHandler: tasksHandler,
	})
	_, err = g.SendTask(context.TODO(), toSend)
	assert.Equal(t, codes.OK, status.Code(err))
	taskToCheck := tasksHandler.Get(int(expectedPairId))
	assert.Equal(t, expectedResult, taskToCheck.GetResult())
}

func TestSendTask(t *testing.T) {
	t.Run("NotFoundCode", testSendTaskNotFoundCode)
	t.Run("AbortedCode", testSendTaskAbortedCode)
	t.Run("OkCode", testSendTaskOkCode)
}

func testRegisterHandlerNewUser(t *testing.T) {
	var err error
	t.Cleanup(func() {
		err = db.Flush()
		if err != nil {
			t.Fatal(err)
		}
	})
	db = callStubDbFabric()
	var (
		requestsToTest    = []*UserStub{{Login: "hhh", Password: "qwertyqwerty"}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCasesHandler[*UserStub, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/register",
			ExpectedHttpCode: http.StatusOK}
	)
	testThroughHttpHandler(registerHandler, t, commonHttpCase, defaultCmpFunc)
}

func TestRegisterHandler(t *testing.T) {
	t.Run("NewUser", testRegisterHandlerNewUser)
	//	t.Run("RegisteredUser", )
}

func TestLoginHandler(t *testing.T) {
	var (
		unregisteredUser = &UserStub{
			Login:    "hhh123",
			Password: "qwertyqwerty",
		}
		registeredUserWithCorrectPassword = &UserStub{
			Login:    "hhh",
			Password: "qwertyqwerty",
		}
		registeredUserWithWrongPassword = &UserStub{
			Login:    registeredUserWithCorrectPassword.Login,
			Password: "asdasdsad",
		}
	)
	db = callStubDbWithRegisteredUserFabric(*registeredUserWithCorrectPassword, *registeredUserWithCorrectPassword)

	t.Run("UnregisteredUserLogin", func(t *testing.T) {
		var (
			requestsToTest    = []*UserStub{unregisteredUser}
			expectedResponses = []backend.EmptyJson{{}}
			commonHttpCase    = backend.HttpCasesHandler[*UserStub, backend.EmptyJson]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/login",
				ExpectedHttpCode: http.StatusUnauthorized}
		)
		testThroughHttpHandler(loginHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("RegisteredUserLoginWithWrongPassword", func(t *testing.T) {
		var (
			requestsToTest    = []*UserStub{registeredUserWithWrongPassword}
			expectedResponses = []*backend.EmptyJson{{}}
			commonHttpCase    = backend.HttpCasesHandler[*UserStub, *backend.EmptyJson]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/login",
				ExpectedHttpCode: http.StatusUnauthorized}
		)
		testThroughHttpHandler(loginHandler, t, commonHttpCase, defaultCmpFunc)
	})
	t.Run("RegisteredUserLoginWithCorrectPassword", func(t *testing.T) {
		var (
			requestsToTest                 = []*UserStub{registeredUserWithCorrectPassword}
			expectedNonIdempotentInstances = []*userForJwtToken{{ExpectedUser: registeredUserWithCorrectPassword}}
			commonHttpCase                 = backend.HttpCasesHandler[*UserStub, *userForJwtToken]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedNonIdempotentInstances, HttpMethod: "POST", UrlTarget: "/api/v1/login",
				ExpectedHttpCode: http.StatusOK}
		)
		testByStructCompareThroughHttpHandler(loginHandler, t, commonHttpCase)
	})
	//t.Run("AuthenticatedUser", func(t *testing.T) {
	//})
}
