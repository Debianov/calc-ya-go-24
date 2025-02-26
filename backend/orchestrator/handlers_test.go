package orchestrator

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var compareTemplate = "ожидается %s, получен %s"
var caseDebugInfoTemplate = "(индекс случая — %d, %s)"

func RunTestThroughHandler[K, V backend.JsonPayload](handler func(w http.ResponseWriter, r *http.Request), t *testing.T,
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

func Test200CalcHandler(t *testing.T) {
	var (
		requestsToTest = []backend.RequestJson{{"2+2*4"}, {"4*2+3"}, {"8+2/3"},
			{"8+3/4*(110+43)-54"}, {""}, {"12"}}
		expectedResponses = []backend.OKJson{{10}, {11}, {8.666666666666666}, {68.75},
			{0}, {12}}
		commonCase = backend.Cases[backend.RequestJson, backend.OKJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusOK}
	)
	RunTestThroughHandler(calcHandler, t, commonCase)
}

func Test422CalcHandler(t *testing.T) {
	var (
		requestsToTest = []backend.RequestJson{{"2++2*4"}, {"4*(2+3"}, {"8+2/3)"},
			{"4*()2+3"}}
		expectedResponses = []backend.ErrorJson{{"Expression is not valid"}, {"Expression is not valid"},
			{"Expression is not valid"}, {"Expression is not valid"}}
		commonCase = backend.Cases[backend.RequestJson, backend.ErrorJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	RunTestThroughHandler(calcHandler, t, commonCase)
}

func Test500CalcHandler(t *testing.T) {
	var (
		requestsToTest    = []backend.RequestNilJson{{Expression: nil}}
		expectedResponses = []backend.OKJson{{}}
		commonCase        = backend.Cases[backend.RequestNilJson, backend.OKJson]{RequestsToSend: requestsToTest,
			ExpectedResponses: expectedResponses, HttpMethod: "POST", UrlTarget: "/api/v1/calculate",
			ExpectedHttpCode: http.StatusUnprocessableEntity}
	)
	RunTestThroughHandler(calcHandler, t, commonCase)
}

//func TestWriteExpressionValidError(t *testing.T) {
//	var (
//		w                   = httptest.NewRecorder()
//		expectedErrResponse = &backend.ErrorJson{Error: "Expression is not valid"}
//		currentErrResponse  backend.ErrorJson
//		buf                 *bytes.Buffer
//		err                 error
//	)
//	writeExpressionValidError(w)
//	buf = w.Body
//	err = json.Unmarshal(buf.Bytes(), &currentErrResponse)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if w.Code != 422 {
//		t.Errorf("ожидается код 422, получен %d", w.Code)
//	}
//	if expectedErrResponse.Error != currentErrResponse.Error {
//		t.Errorf(compareTemplate, expectedErrResponse.Error, expectedErrResponse.Error)
//	}
//}

func TestGoodPanicMiddleware(t *testing.T) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", mockHandlerWithoutPanic)
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

func mockHandlerWithoutPanic(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	return
}

func TestBadPanicMiddleware(t *testing.T) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", mockHandlerWithPanic)
	middlewareHandler := panicMiddleware(mux)
	var (
		w                   = httptest.NewRecorder()
		mockReader          = bytes.NewReader(nil)
		req                 = httptest.NewRequest("GET", "/api/v1/calculate", mockReader)
		expectedErrResponse = &backend.ErrorJson{Error: "Internal server error"}
		gottenErrResponse   backend.ErrorJson
		err                 error
	)
	middlewareHandler.ServeHTTP(w, req)
	err = json.Unmarshal(w.Body.Bytes(), &gottenErrResponse)
	if err != nil {
		t.Fatal(err)
	}
	if 500 != w.Code {
		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
	}
	if expectedErrResponse.Error != gottenErrResponse.Error {
		t.Errorf(compareTemplate, expectedErrResponse.Error, gottenErrResponse.Error)
	}
}

func mockHandlerWithPanic(_ http.ResponseWriter, _ *http.Request) {
	panic(errors.New("ААААААА!!!!"))
}

func TestInternalServerErrorHandler(t *testing.T) {
	var (
		w                   = httptest.NewRecorder()
		expectedErrResponse = &backend.ErrorJson{Error: "Internal server error"}
		gottenErrResponse   backend.ErrorJson
		err                 error
	)
	writeInternalServerError(w)
	err = json.Unmarshal(w.Body.Bytes(), &gottenErrResponse)
	if err != nil {
		t.Fatal(err)
	}
	if 500 != w.Code {
		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
	}
	if expectedErrResponse.Error != gottenErrResponse.Error {
		t.Errorf(compareTemplate, expectedErrResponse.Error, gottenErrResponse.Error)
	}
}

func TestGoodGetHandler(t *testing.T) {
	var (
		handler          = getHandler()
		w                = httptest.NewRecorder()
		reqJson          = backend.RequestJson{Expression: "23+21/3*123"}
		reqJsonInByte    []byte
		reqToSend        *http.Request
		expectedResponse = backend.OKJson{Result: 884}
		gottenResponse   backend.OKJson
		err              error
	)
	reqJsonInByte, err = json.Marshal(reqJson)
	if err != nil {
		t.Fatal(err)
	}
	reqToSend, err = http.NewRequest("POST", "/api/v1/calculate", bytes.NewReader(reqJsonInByte))
	if err != nil {
		t.Fatal(err)
	}
	handler.ServeHTTP(w, reqToSend)
	err = json.Unmarshal(w.Body.Bytes(), &gottenResponse)
	if expectedResponse.Result != gottenResponse.Result {
		t.Errorf(compareTemplate, strconv.Itoa(int(expectedResponse.Result)), strconv.Itoa(int(gottenResponse.Result)))
	}
	if 200 != w.Code {
		t.Errorf(compareTemplate, "200", strconv.Itoa(w.Code))
	}
}

/*
TestBadGetHandler тестирует, что, в общем, цепочка handler-ов в getHandler функции построена верно.
*/
func TestBadGetHandler(t *testing.T) {
	var handler = getHandler()

	requestsToTest := []backend.JsonPayload{backend.RequestJson{"232+)"}}
	expectedResponses := []backend.ErrorJson{{Error: "Expression is not valid"}}
	RunTestThroughHandler(handler.ServeHTTP, t, requestsToTest, expectedResponses, 422)

	requestsToTest = []backend.JsonPayload{backend.RequestNilJson{Expression: nil}}
	expectedResponses = []backend.ErrorJson{{Error: "Internal server error"}}
	RunTestThroughHandler(handler.ServeHTTP, t, requestsToTest, expectedResponses, 500)
}
