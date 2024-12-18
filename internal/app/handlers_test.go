package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var compareErrorTemplate = "ожидается %s error, получен %s error"

func TestErrorHandler(t *testing.T) {
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
		t.Fatalf(compareErrorTemplate, expectedErrResponse.Error, expectedErrResponse.Error)
	}
}

type byteCase struct {
	toOutput []byte
	expected []byte
}

func TestGoodCalcHandler(t *testing.T) {
	var (
		requestsToTest = []RequestJson{{"2+2*4"}, {"4*2+3"}, {"8+2/3"},
			{"8+3/4*(110+43)-54"}}
		expectedResponses = []OKJson{{10}, {11}, {8.666666666666666}, {68.75}}
	)
	RunThroughCalcHandler(t, requestsToTest, expectedResponses)
}

func TestBadCalcHandler(t *testing.T) {
	var (
		requestsToTest = []RequestJson{{"2++2*4"}, {"4*(2+3"}, {"8+2/3)"},
			{"4*()2+3"}}
		expectedResponses = []ErrorJson{{"Expression is not valid"}, {"Expression is not valid"},
			{"Expression is not valid"}, {"Expression is not valid"}}
	)
	RunThroughCalcHandler(t, requestsToTest, expectedResponses)
}

func RunThroughCalcHandler[K, V JsonPayload](t *testing.T, requestsToSend []K, expectedResponses []V) {
	var (
		cases []byteCase
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
		reader = bytes.NewReader(testCase.toOutput)
		req = httptest.NewRequest("POST", "/api/v1/calculate", reader)
		CalcHandler(w, req)
		if bytes.Compare(testCase.expected, w.Body.Bytes()) != 0 {
			t.Fatalf(compareErrorTemplate+" "+"(индекс случая — %d, %s)", testCase.expected, w.Body.Bytes(), ind, testCase)
		}
	}
}

func convertToByteCases[K, V JsonPayload](reqs []K, resps []V) (result []byteCase, err error) {
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
		result = append(result, byteCase{reqBuf, respBuf})
	}
	return
}
