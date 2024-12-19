package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var compareTemplate = "ожидается %s, получен %s"
var caseDebugInfoTemplate = "(индекс случая — %d, %s)"

func Test200CalcHandler(t *testing.T) {
	var (
		requestsToTest = []RequestJson{{"2+2*4"}, {"4*2+3"}, {"8+2/3"},
			{"8+3/4*(110+43)-54"}}
		expectedResponses = []OKJson{{10}, {11}, {8.666666666666666}, {68.75}}
	)
	RunThroughCalcHandler(t, requestsToTest, expectedResponses, 200)
}

func Test422CalcHandler(t *testing.T) {
	var (
		requestsToTest = []RequestJson{{"2++2*4"}, {"4*(2+3"}, {"8+2/3)"},
			{"4*()2+3"}}
		expectedResponses = []ErrorJson{{"Expression is not valid"}, {"Expression is not valid"},
			{"Expression is not valid"}, {"Expression is not valid"}}
	)
	RunThroughCalcHandler(t, requestsToTest, expectedResponses, 422)
}

func RunThroughCalcHandler[K, V JsonPayload](t *testing.T, requestsToSend []K, expectedResponses []V, expectedHttpCode int) {
	var (
		cases []pkg.ByteCase
		err   error
	)
	cases, err = convertToByteCases(requestsToSend, expectedResponses)
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
		req = httptest.NewRequest("POST", "/api/v1/calculate", reader)
		CalcHandler(w, req)
		if bytes.Compare(testCase.Expected, w.Body.Bytes()) != 0 {
			t.Fatalf(compareTemplate+" "+caseDebugInfoTemplate, testCase.Expected, w.Body.Bytes(), ind, testCase)
		}
		if w.Code != expectedHttpCode {
			t.Fatalf(compareTemplate+" "+caseDebugInfoTemplate, strconv.Itoa(expectedHttpCode), strconv.Itoa(w.Code),
				ind, testCase)
		}
	}
}

func convertToByteCases[K, V JsonPayload](reqs []K, resps []V) (result []pkg.ByteCase, err error) {
	if len(reqs) != len(resps) {
		err = errors.New("reqs и resps должны быть одной длины")
		return
	}
	for ind := 0; ind < len(reqs); ind++ {
		var (
			reqBuf  []byte
			respBuf []byte
		)
		reqBuf, err = reqs[ind].Marshal()
		if err != nil {
			return
		}
		respBuf, err = resps[ind].Marshal()
		if err != nil {
			return
		}
		result = append(result, pkg.ByteCase{ToOutput: reqBuf, Expected: respBuf})
	}
	return
}

func TestExpressionValidErrorHandler(t *testing.T) {
	var (
		w                   = httptest.NewRecorder()
		expectedErrResponse = &ErrorJson{Error: "Expression is not valid"}
		currentErrResponse  ErrorJson
		buf                 *bytes.Buffer
		err                 error
	)
	expressionValidErrorHandler(w)
	if w.Code != 422 {
		t.Fatalf("ожидается код 422, получен %d", w.Code)
	}
	buf = w.Body
	err = json.Unmarshal(buf.Bytes(), &currentErrResponse)
	if err != nil {
		t.Fatal(err)
	}
	if currentErrResponse.Error != expectedErrResponse.Error {
		t.Fatalf(compareTemplate, expectedErrResponse.Error, expectedErrResponse.Error)
	}
}

func TestBadPanicMiddleware(t *testing.T) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", mockHandlerWithPanic)
	middlewareHandler := PanicMiddleware(mux)
	var (
		w                   = httptest.NewRecorder()
		mockReader          = bytes.NewReader(nil)
		req                 = httptest.NewRequest("GET", "/api/v1/calculate", mockReader)
		expectedErrResponse = &ErrorJson{Error: "Internal server error"}
		gottenErrResponse   ErrorJson
		err                 error
	)
	middlewareHandler.ServeHTTP(w, req)
	err = json.Unmarshal(w.Body.Bytes(), &gottenErrResponse)
	if err != nil {
		t.Error(err)
	}
	if expectedErrResponse.Error != gottenErrResponse.Error {
		t.Errorf(compareTemplate, expectedErrResponse.Error, gottenErrResponse.Error)
	}
	if w.Code != 500 {
		t.Errorf(compareTemplate, "500", strconv.Itoa(w.Code))
	}
}

func mockHandlerWithPanic(w http.ResponseWriter, r *http.Request) {
	panic(errors.New("ААААААА!!!!"))
}
