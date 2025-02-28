package orchestrator

import (
	"bytes"
	"github.com/Debianov/calc-ya-go-24/backend"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var compareTemplate = "ожидается \"%s\", получен \"%s\""
var caseDebugInfoTemplate = "(индекс случая — %d, параметры случая — %s)"

func runTestThroughHandler[K, V backend.JsonPayload](handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
	testCases backend.Cases[K, V]) {
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
			reader *bytes.Reader
			req    *http.Request
		)
		reader = bytes.NewReader(testCase.ToOutput)
		req = httptest.NewRequest(testCases.HttpMethod, testCases.UrlTarget, reader)
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

func testCalcHandler200(t *testing.T) {
	var (
		requestsToTest = []backend.RequestJson{{"2+2*4"}, {"4*2+3"}, {"8+2/3"},
			{"8+3/4*(110+43)-54"}, {""}, {"12"}}
		expectedResponses = []*backend.Expression{{ID: 0}, {ID: 1}, {ID: 2}, {ID: 3},
			{ID: 4}, {ID: 5}}
		commonHttpCase = backend.Cases[backend.RequestJson, *backend.Expression]{RequestsToSend: requestsToTest,
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
		commonHttpCase    = backend.Cases[backend.RequestJson, backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	runTestThroughHandler(calcHandler, t, commonHttpCase)
}

func testCalcHandlerGet(t *testing.T) {
	var (
		requestsToTest    = []backend.RequestJson{{"2+2*4"}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.Cases[backend.RequestJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(calcHandler, t, commonHttpCase)
}

func TestCalcHandler(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListFabric()
	})
	t.Run("TestCalcHandler200", testCalcHandler200)
	t.Run("TestCalcHandler422", testCalcHandler422)
	t.Run("TestCalcHandlerGet", testCalcHandlerGet)
}

func testExpressionHandler200(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0},
			{ID: 1, Status: backend.Completed, Result: 432}, {ID: 2, Status: backend.Cancelled, Result: 0}, {ID: 3,
				Status: backend.NoReadyTasks, Result: 0}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.Expressions{{expectedExpressions}}
		commonHttpCase    = backend.Cases[backend.EmptyJson, *backend.Expressions]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)
}

func testExpressionHandlerPost(t *testing.T) {
	t.Cleanup(func() {
		exprsList = backend.ExpressionListFabric()
	})
	var (
		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0}}
	)
	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.EmptyJson{{}}
		commonHttpCase    = backend.Cases[backend.EmptyJson, *backend.EmptyJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)
}

func testExpressionHandlerEmpty(t *testing.T) {
	exprsList = backend.ExpressionListFabric() // пустой список выражений
	var (
		requestsToTest    = []backend.EmptyJson{{}}
		expectedResponses = []*backend.Expressions{{Expressions: make([]*backend.Expression, 0)}}
		commonHttpCase    = backend.Cases[backend.EmptyJson, *backend.Expressions]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "GET", UrlTarget: "/api/v1/expressions",
			ExpectedHttpCode: http.StatusOK}
	)
	runTestThroughHandler(expressionsHandler, t, commonHttpCase)

}

func TestExpressionHandler(t *testing.T) {
	t.Run("TestExpressionHandler200", testExpressionHandler200)
	t.Run("TestExpressionHandlerPost", testExpressionHandlerPost)
	t.Run("TestExpressionHadnlerEmpty", testExpressionHandlerEmpty)
}

//func expressionIdHandler200(t *testing.T) {
//
//}
//
//func TestExpressionIdHandler(t *testing.T) {
//	t.Cleanup(func() {
//		exprsList = backend.ExpressionListFabric()
//	})
//	var (
//		expectedExpressions = []*backend.Expression{{ID: 0, Status: backend.Ready, Result: 0}}
//	)
//	exprsList = backend.ExpressionListFabricWithElements(expectedExpressions)
//	t.Run("TestExpressionIdHandler", expressionIdHandler200)
//}

//func TestGoodPanicMiddleware(t *testing.T) {
//	var mux = http.NewServeMux()
//	mux.HandleFunc("/api/v1/calculate", mockHandlerWithoutPanic)
//	var (
//		middlewareHandler = panicMiddleware(mux)
//		w                 = httptest.NewRecorder()
//		mockReader        = bytes.NewReader(nil)
//		req               = httptest.NewRequest("POST", "/api/v1/calculate", mockReader)
//	)
//	middlewareHandler.ServeHTTP(w, req)
//	if 200 != w.Code {
//		t.Errorf(compareTemplate, "200", strconv.Itoa(w.Code))
//	}
//}
//
//func mockHandlerWithoutPanic(w http.ResponseWriter, _ *http.Request) {
//	w.WriteHeader(200)
//	return
//}
//
//func TestBadPanicMiddleware(t *testing.T) {
//	var mux = http.NewServeMux()
//	mux.HandleFunc("/api/v1/calculate", mockHandlerWithPanic)
//	middlewareHandler := panicMiddleware(mux)
//	var (
//		w                   = httptest.NewRecorder()
//		mockReader          = bytes.NewReader(nil)
//		req                 = httptest.NewRequest("GET", "/api/v1/calculate", mockReader)
//		expectedErrResponse = &backend.ErrorJson{Error: "Internal server error"}
//		gottenErrResponse   backend.ErrorJson
//		err                 error
//	)
//	middlewareHandler.ServeHTTP(w, req)
//	err = json.Unmarshal(w.Body.Bytes(), &gottenErrResponse)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if 500 != w.Code {
//		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
//	}
//	if expectedErrResponse.Error != gottenErrResponse.Error {
//		t.Errorf(compareTemplate, expectedErrResponse.Error, gottenErrResponse.Error)
//	}
//}
//
//func mockHandlerWithPanic(_ http.ResponseWriter, _ *http.Request) {
//	panic(errors.New("ААААААА!!!!"))
//}
//
//func TestInternalServerErrorHandler(t *testing.T) {
//	var (
//		w                   = httptest.NewRecorder()
//		expectedErrResponse = &backend.ErrorJson{Error: "Internal server error"}
//		gottenErrResponse   backend.ErrorJson
//		err                 error
//	)
//	writeInternalServerError(w)
//	err = json.Unmarshal(w.Body.Bytes(), &gottenErrResponse)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if 500 != w.Code {
//		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
//	}
//	if expectedErrResponse.Error != gottenErrResponse.Error {
//		t.Errorf(compareTemplate, expectedErrResponse.Error, gottenErrResponse.Error)
//	}
//}
//
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

///*
//TestBadGetHandler тестирует, что, в общем, цепочка handler-ов в getHandler функции построена верно.
//*/
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
