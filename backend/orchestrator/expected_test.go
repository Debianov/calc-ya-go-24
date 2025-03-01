// Тестирование случаев, предусмотренных ТЗ.

package orchestrator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Debianov/calc-ya-go-24/backend"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

var compareTemplate = "ожидается \"%s\", получен \"%s\""
var caseDebugInfoTemplate = "(индекс случая — %d, параметры случая — %s)"

// runTestThroughHandler запускает все тесты через handler, используя параметры testCases.
// Генерируемый запрос всегда отправляется с заголовком "Content-Type": "application/json".
func runTestThroughHandler[K, V backend.JsonPayload](handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	testCases backend.HttpCases[K, V]) {
	var (
		cases []backend.ByteCase
		err   error
	)
	cases, err = backend.ConvertToByteCases(testCases.RequestsToSend, testCases.ExpectedResponses)
	if err != nil {
		t.Fatal(err)
	}
	for ind, testCase := range cases {
		var (
			w      = httptest.NewRecorder()
			reader = bytes.NewReader(testCase.ToOutput)
			req    = httptest.NewRequest(testCases.HttpMethod, testCases.UrlTarget, reader)
		)
		req.Header.Set("Content-Type", "application/json")
		handler(w, req)
		if testCases.ExpectedHttpCode != w.Code {
			t.Errorf(compareTemplate+" "+caseDebugInfoTemplate, strconv.Itoa(testCases.ExpectedHttpCode),
				strconv.Itoa(w.Code), ind, testCase)
		}
		if bytes.Compare(testCase.Expected, w.Body.Bytes()) != 0 {
			t.Errorf(compareTemplate+" "+caseDebugInfoTemplate, testCase.Expected, w.Body.Bytes(), ind, testCase)
		}
	}
}

// runTestThroughServeMux работает также, как и runTestThroughHandler, но с обёрткой handler-а в http.ServerMux.
// Необходимо для тестирования некоторых handler-ов, которые вызывают методы, связанные с парсингом URL в запросах
// (например, request.PathValue). Парсинг происходит только при вызове http.ServerMux
// (https://pkg.go.dev/net/http#ServeMux).
func runTestThroughServeMux[K, V backend.JsonPayload](handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	testCases backend.ServerMuxHttpCases[K, V]) {
	var (
		cases []backend.ByteCase
		err   error
	)
	cases, err = backend.ConvertToByteCases(testCases.RequestsToSend, testCases.ExpectedResponses)
	if err != nil {
		t.Fatal(err)
	}
	for ind, testCase := range cases {
		var (
			w         = httptest.NewRecorder()
			reader    = bytes.NewReader(testCase.ToOutput)
			req       = httptest.NewRequest(testCases.HttpMethod, testCases.UrlTarget, reader)
			serverMux = http.NewServeMux()
		)
		req.Header.Set("Content-Type", "application/json")
		serverMux.HandleFunc(testCases.UrlTemplate, handler)
		serverMux.ServeHTTP(w, req)
		if testCases.ExpectedHttpCode != w.Code {
			t.Errorf(compareTemplate+" "+caseDebugInfoTemplate, strconv.Itoa(testCases.ExpectedHttpCode),
				strconv.Itoa(w.Code), ind, testCase)
		}
		if bytes.Compare(testCase.Expected, w.Body.Bytes()) != 0 {
			t.Errorf(compareTemplate+" "+caseDebugInfoTemplate, testCase.Expected, w.Body.Bytes(), ind, testCase)
		}
	}
}

func testCalcHandler200(t *testing.T) {
	// TODO проверка, что выражение добавлено
	var (
		requestsToTest = []backend.RequestJson{{"2+2*4"}, {"4*2+3"}, {"8+2/3"},
			{"8+3/4*(110+43)-54"}, {""}, {"12"}}
		expectedResponses = []*backend.Expression{{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3},
			{ID: 4}, {ID: 5}}
		commonHttpCase = backend.HttpCases[backend.RequestJson, *backend.Expression]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(calcHandler, t, commonHttpCase)
}

func testCalcHandler422(t *testing.T) {
	var (
		requestsToTest = []backend.RequestJson{{"2++2*4"}, {"4*(2+3"}, {"8+2/3)"},
			{"4*()2+3"}}
		expectedResponses = []backend.EmptyJson{{}, {}, {}, {}}
		commonHttpCase    = backend.HttpCases[backend.RequestJson, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	runTestThroughHandler(calcHandler, t, commonHttpCase)
}

func testCalcHandlerGet(t *testing.T) {
	var (
		requestsToTest    = []backend.RequestJson{{"2+2*4"}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[backend.RequestJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(calcHandler, t, commonHttpCase)
}

func TestCalcHandler(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	t.Run("TestCalcHandler200", testCalcHandler200)
	t.Run("TestCalcHandler422", testCalcHandler422)
	t.Run("TestCalcHandlerGet", testCalcHandlerGet)
}

func testExpressionsHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0},
			{ID: 1, Status: backend.Completed, Result: 432}, {ID: 2, Status: backend.Cancelled, Result: 0},
			{ID: 3, Status: backend.NoReadyTasks, Result: 0}, {ID: 4, Status: backend.Completed, Result: -2345}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.ExpressionsJsonTitle{{expectedExpressions}}
		commonHttpCase    = backend.HttpCases[backend.EmptyJson, *backend.ExpressionsJsonTitle]{RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)
}

func testExpressionsHandlerPost(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[backend.EmptyJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)
}

func testExpressionsHandlerEmpty(t *testing.T) {
	exprsList = backend.ExpressionListEmptyFabric()
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.ExpressionsJsonTitle{{Expressions: make([]*backend.Expression, 0)}}
		commonHttpCase    = backend.HttpCases[backend.EmptyJson, *backend.ExpressionsJsonTitle]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)

}

func TestExpressionHandler(t *testing.T) {
	t.Run("TestExpressionsHandler200", testExpressionsHandler200)
	t.Run("TestExpressionsHandlerPost", testExpressionsHandlerPost)
	t.Run("TestExpressionsHandlerEmpty", testExpressionsHandlerEmpty)
}

func testExpressionIdHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0},
			{ID: 1, Status: backend.Completed, Result: 431}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	for ind, expExpr := range expectedExpressions {
		t.Run(fmt.Sprintf("ExpressionId%d", ind), func(t *testing.T) {
			var (
				requestsToTest    = []backend.EmptyJson{{}}
				expectedResponses = []*backend.ExpressionJsonTitle{{expExpr}}
				serverMuxHttpCase = backend.ServerMuxHttpCases[backend.EmptyJson, *backend.ExpressionJsonTitle]{
					RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
					UrlTemplate: "/api/v1/expressions/{ID}", UrlTarget: fmt.Sprintf("/api/v1/expressions/%d", ind),
					ExpectedHttpCode: http.StatusOK}
			)
			runTestThroughServeMux(expressionIdHandler, t, serverMuxHttpCase)
		})
	}
}

func testExpressionIdHandler404(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCases[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
			UrlTemplate: "/api/v1/expressions/{ID}", UrlTarget: "/api/v1/expressions/1",
			ExpectedHttpCode: http.StatusNotFound}
	)
	runTestThroughServeMux(expressionIdHandler, t, serverMuxHttpCase)
}

func testExpressionIdHandlerPost(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCases[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "POST",
			UrlTemplate: "/api/v1/expressions/{ID}", UrlTarget: "/api/v1/expressions/0",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughServeMux(expressionIdHandler, t, serverMuxHttpCase)
}

func testExpressionIdHandlerEmpty(t *testing.T) {
	exprsList = backend.ExpressionListEmptyFabric()
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		serverMuxHttpCase = backend.ServerMuxHttpCases[backend.EmptyJson, *backend.EmptyJson]{
			RequestsToSend: requestsToTest, ExpectedResponses: expectedResponses, HttpMethod: "GET",
			UrlTemplate: "/api/v1/expressions/{ID}", UrlTarget: "/api/v1/expressions/0",
			ExpectedHttpCode: http.StatusNotFound}
	)
	runTestThroughServeMux(expressionIdHandler, t, serverMuxHttpCase)
}

func TestExpressionIdHandler(t *testing.T) {
	t.Run("TestExpressionIdHandler200", testExpressionIdHandler200)
	t.Run("TestExpressionIdHandler404", testExpressionIdHandler404)
	t.Run("TestExpressionIdHandlerPost", testExpressionIdHandlerPost)
	t.Run("TestExpressionIdHandlerEmpty", testExpressionIdHandlerEmpty)
}

func testTaskGetHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	exprsList.ExprFabricAdd([]string{"2", "3", "*"})
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.TaskToSend{{Task: &backend.Task{
			PairID:        0,
			Arg1:          2,
			Arg2:          3,
			Operation:     "*",
			OperationTime: 1 * time.Second,
		}}}
		commonHttpCase = backend.HttpCases[backend.EmptyJson, *backend.TaskToSend]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(taskHandler, t, commonHttpCase)
}

func testTaskGetHandlerEmpty404(t *testing.T) {
	exprsList = backend.ExpressionListEmptyFabric()
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[backend.EmptyJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusNotFound}
	)
	runTestThroughHandler(taskHandler, t, commonHttpCase)
}

func testTaskGetHandler404(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	exprsList.ExprFabricAdd([]string{"2", "3", "*"})
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.TaskToSend{{Task: &backend.Task{
			PairID:        0,
			Arg1:          2,
			Arg2:          3,
			Operation:     "*",
			OperationTime: 1 * time.Second,
		}}}
		requestsToTest2    = []backend.EmptyJson{{}}
		expectedResponses2 = []*backend.EmptyJson{{}}
		commonHttpCase     = backend.HttpCases[backend.EmptyJson, *backend.TaskToSend]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusOK}
		commonHttpCase2 = backend.HttpCases[backend.EmptyJson, *backend.EmptyJson]{RequestsToSend: requestsToTest2,
			ExpectedResponses: expectedResponses2, HttpMethod: "GET", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusNotFound}
	)
	runTestThroughHandler(taskHandler, t, commonHttpCase)
	runTestThroughHandler(taskHandler, t, commonHttpCase2)
}

func testTaskPostHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListEmptyFabric()
	})
	exprsList.ExprFabricAdd([]string{"2", "3", "*"})
	var (
		requestsToTest    = []*backend.AgentResult{{ID: 0, Result: 6}}
		expectedResponses = []backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[*backend.AgentResult, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusOK}
	)
	stubExpr := exprsList.GetReadyExpr()
	stubExpr.FabricReadyExprSendTask()
	stubTask := stubExpr.TasksHandler.Get(0)
	stubTask.ChangeStatus(backend.Sent)

	runTestThroughHandler(taskHandler, t, commonHttpCase)

	if stubExpr.Result != 6 {
		t.Errorf("Ожидается Result %d по Expression %d, получен %d", 6, stubExpr.ID, stubExpr.Result)
	}
}

func testTaskPostHandler404(t *testing.T) {
	exprsList = backend.ExpressionListEmptyFabric()
	var (
		requestsToTest    = []*backend.AgentResult{{ID: 0, Result: 6}}
		expectedResponses = []backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[*backend.AgentResult, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusNotFound}
	)
	runTestThroughHandler(taskHandler, t, commonHttpCase)
}

type RandomJson struct {
	Hey   int `json:"hey"`
	Issue int `json:"issue"`
}

func (r *RandomJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}

func testTaskPostHandler422(t *testing.T) {
	exprsList = backend.ExpressionListEmptyFabric()
	var (
		requestsToTest    = []*RandomJson{{Hey: 0, Issue: 6}}
		expectedResponses = []backend.EmptyJson{{}}
		commonHttpCase    = backend.HttpCases[*RandomJson, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/internal/task",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	runTestThroughHandler(taskHandler, t, commonHttpCase)
}

func TestTaskHandler(t *testing.T) {
	t.Run("TestTaskGetHandler200", testTaskGetHandler200)
	t.Run("TestTaskGetHandlerEmpty404", testTaskGetHandlerEmpty404)
	t.Run("TestTaskGetHandler404", testTaskGetHandler404)
	t.Run("TestTaskPostHandler200", testTaskPostHandler200)
	t.Run("TestTaskPostHandler404", testTaskPostHandler404)
	t.Run("TestTaskPostHandler422", testTaskPostHandler422)
}

func TestGoodPanicMiddleware(t *testing.T) {
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

func TestBadPanicMiddleware(t *testing.T) {
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

//func TestGoodGetHandler(t *testing.T) {
//	var (
//		handler          = getHandler()
//		w                = httptest.NewRecorder()
//		reqJson          = backend.RequestJson{Expression: "23+21/3*123"}
//		reqJsonInByte    []byte
//		reqToSend        *http.Request
//		expectedResponse = backend.OKJson{Result: 884}
//		gottenResponse   backend.OKJson
//		err              error
//	)
//	reqJsonInByte, err = json.Marshal(reqJson)
//	if err != nil {
//		t.Fatal(err)
//	}
//	reqToSend, err = http.NewRequest("POST", "/api/v1/calculate", bytes.NewReader(reqJsonInByte))
//	if err != nil {
//		t.Fatal(err)
//	}
//	handler.ServeHTTP(w, reqToSend)
//	err = json.Unmarshal(w.Body.Bytes(), &gottenResponse)
//	if expectedResponse.Result != gottenResponse.Result {
//		t.Errorf(compareTemplate, strconv.Itoa(int(expectedResponse.Result)), strconv.Itoa(int(gottenResponse.Result)))
//	}
//	if 200 != w.Code {
//		t.Errorf(compareTemplate, "200", strconv.Itoa(w.Code))
//	}
//}

//func TestBadGetHandler(t *testing.T) {
//	var handler = getHandler()
//
//	requestsToTest := []backend.JsonPayload{backend.RequestJson{"232+)"}}
//	expectedResponses := []backend.ErrorJson{{Error: "Expression is not valid"}}
//	runTestThroughHandler(handler.ServeHTTP, t, requestsToTest, expectedResponses, 422)
//
//	requestsToTest = []backend.JsonPayload{backend.RequestNilJson{Expression: nil}}
//	expectedResponses = []backend.ErrorJson{{Error: "Internal server error"}}
//	runTestThroughHandler(handler.ServeHTTP, t, requestsToTest, expectedResponses, 500)
//}
