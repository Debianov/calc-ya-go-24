// Тестирование случаев, предусмотренных ТЗ.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Debianov/calc-ya-go-24/backend"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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

func testCalcHandler201(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		requestsToTest    = []*backend.RequestJson{{"2+2*4"}, {"4*2+3*5"}}
		expectedResponses = []*backend.ExpressionJsonStub{{ID: 0}, {ID: 1}}
		commonHttpCase    = backend.HttpCasesHandler[*backend.RequestJson, *backend.ExpressionJsonStub]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusCreated}
	)
	testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)

	var (
		expectedLen               = len(expectedResponses)
		expectedTasksForFirstExpr = []backend.Task{*backend.CallTaskFabric(0, 2, 4, "*",
			backend.ReadyToCalc), *backend.CallTaskFabric(1, nil, 2, "+",
			backend.WaitingOtherTasks)}
		expectedTasksForSecondExpr = []backend.Task{*backend.CallTaskFabric(2, 4, 2, "*",
			backend.ReadyToCalc), *backend.CallTaskFabric(3, 3, 5, "*",
			backend.ReadyToCalc), *backend.CallTaskFabric(5, nil, nil, "+",
			backend.WaitingOtherTasks)}
		expectedTasks = [][]backend.Task{expectedTasksForFirstExpr, expectedTasksForSecondExpr}
	)
	exprs := exprsList.GetAllExprs()
	slices.SortFunc(exprs, func(expression backend.CommonExpression, expression2 backend.CommonExpression) int {
		if expression.GetId() >= expression2.GetId() {
			return 0
		} else {
			return -1
		}
	})
	assert.Equal(t, len(exprs), expectedLen)
	for exprInd, expr := range exprs {
		var tasksListLen = expr.GetTasksHandler().Len()
		for taskInd := 0; taskInd < tasksListLen; taskInd++ {
			var (
				v    interface{}
				expV interface{}
			)
			task := expr.GetTasksHandler().Get(taskInd)
			assert.Equal(t, task.GetPairId(), expectedTasks[exprInd][taskInd].GetPairId())
			v, _ = task.GetArg1()
			expV, _ = expectedTasks[exprInd][taskInd].GetArg1()
			assert.Equal(t, expV, v)
			v, _ = task.GetArg2()
			expV, _ = expectedTasks[exprInd][taskInd].GetArg2()
			assert.Equal(t, expV, v)
			v = task.GetOperation()
			expV = expectedTasks[exprInd][taskInd].GetOperation()
			assert.Equal(t, expV, v)
		}
	}
}

func testCalcHandler422(t *testing.T) {
	var (
		requestsToTest = []backend.RequestJson{{"2++2*4"}, {"4*(2+3"}, {"8+2/3)"},
			{"4*()2+3"}}
		expectedResponses = []backend.EmptyJson{{}, {}, {}, {}}
		commonHttpCase    = backend.HttpCasesHandler[backend.RequestJson, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)
}

func testCalcHandlerGet(t *testing.T) {
	var (
		requestsToTest    = []backend.RequestJson{{"2+2*4"}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCasesHandler[backend.RequestJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusNotFound}
	)
	testThroughHttpHandler(calcHandler, t, commonHttpCase, defaultCmpFunc)
}

func TestCalcHandler(t *testing.T) {
	t.Run("201Code", testCalcHandler201)
	t.Run("422Code", testCalcHandler422)
	t.Run("Get", testCalcHandlerGet)
}

func testExpressionsHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0},
			{Id: 1, Status: backend.Completed, Result: 432}, {Id: 2, Status: backend.Cancelled, Result: 0},
			{Id: 3, Status: backend.NoReadyTasks, Result: 0}, {Id: 4, Status: backend.Completed, Result: -2345}}
	)
	exprsList = callExprsListStubFabric(expectedExpressions...)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.ExpressionsJsonTitleStub{{expectedExpressions}}
		commonHttpCase    = backend.HttpCasesHandler[backend.EmptyJson, *backend.ExpressionsJsonTitleStub]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
}

func testExpressionsHandlerPost(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = callExprsListStubFabric(expectedExpressions...)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCasesHandler[backend.EmptyJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusNotFound}
	)
	testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)
}

func testExpressionsHandlerEmpty(t *testing.T) {
	exprsList = CallEmptyExpressionListFabric()
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.ExpressionsJsonTitle{{Expressions: make([]backend.CommonExpression, 0)}}
		commonHttpCase    = backend.HttpCasesHandler[backend.EmptyJson, *backend.ExpressionsJsonTitle]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	testThroughHttpHandler(expressionsHandler, t, commonHttpCase, defaultCmpFunc)

}

func TestExpressionHandler(t *testing.T) {
	t.Run("200Code", testExpressionsHandler200)
	t.Run("Post", testExpressionsHandlerPost)
	t.Run("Empty", testExpressionsHandlerEmpty)
}

func testExpressionIdHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0},
			{Id: 1, Status: backend.Completed, Result: 431}}
	)
	exprsList = callExprsListStubFabric(expectedExpressions...)
	for ind, expExpr := range expectedExpressions {
		t.Run(fmt.Sprintf("ExpressionId%d", ind), func(t *testing.T) {
			var (
				requestsToTest    = []backend.EmptyJson{{}}
				expectedResponses = []*backend.ExpressionJsonTitleStub{{expExpr}}
				serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[backend.EmptyJson, *backend.ExpressionJsonTitleStub]{
					RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
					UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: fmt.Sprintf("/api/v1/expressions/%d", ind),
					ExpectedHttpCode: http.StatusOK}
			)
			testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
		})
	}
}

func testExpressionIdHandler404(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = callExprsListStubFabric(expectedExpressions...)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
			UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/1",
			ExpectedHttpCode: http.StatusNotFound}
	)
	testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
}

func testExpressionIdHandlerPost(t *testing.T) {
	t.Cleanup(func() {
		exprsList = CallEmptyExpressionListFabric()
	})
	var (
		expectedExpressions = []backend.ExpressionStub{{Id: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = callExprsListStubFabric(expectedExpressions...)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "POST",
			UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
			ExpectedHttpCode: http.StatusNotFound}
	)
	testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
}

func testExpressionIdHandlerEmpty(t *testing.T) {
	exprsList = CallEmptyExpressionListFabric()
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCasesHandler[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
			UrlTemplate: "/api/v1/expressions/{id}", UrlTarget: "/api/v1/expressions/0",
			ExpectedHttpCode: http.StatusNotFound}
	)
	testThroughServeMux(expressionIdHandler, t, serverMuxHttpCase, defaultCmpFunc)
}

func TestExpressionIdHandler(t *testing.T) {
	t.Run("200Code", testExpressionIdHandler200)
	t.Run("404Code", testExpressionIdHandler404)
	t.Run("Post", testExpressionIdHandlerPost)
	t.Run("Empty", testExpressionIdHandlerEmpty)
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
	exprsList = callExprsListStubFabric()
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
	exprsList = callExprsListStubFabric(backend.ExpressionStub{
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
	exprsList = callExprsListStubFabric(backend.ExpressionStub{
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
	exprsList = callExprsListStubFabric()
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
	exprsList = callExprsListStubFabric(backend.ExpressionStub{
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
	exprsList = callExprsListStubFabric(backend.ExpressionStub{
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
			expectedNonIdempotentInstances = []*jwtTokenStub{{ExpectedUser: registeredUserWithCorrectPassword}}
			commonHttpCase                 = backend.HttpCasesHandler[*UserStub, *jwtTokenStub]{RequestsToSend: requestsToTest,
				ExpectedResponses: expectedNonIdempotentInstances, HttpMethod: "POST", UrlTarget: "/api/v1/login",
				ExpectedHttpCode: http.StatusOK}
		)
		testByStructCompareThroughHttpHandler(loginHandler, t, commonHttpCase)
	})
	//t.Run("AuthenticatedUser", func(t *testing.T) {
	//})
}
