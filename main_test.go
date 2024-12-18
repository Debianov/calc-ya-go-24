package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestErrorHandler(t *testing.T) {
	var (
		w                   = httptest.NewRecorder()
		expectedErrResponse = &ErrorResponse{Error: "Expression is not valid"}
		currentErrResponse  ErrorResponse
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
		t.Fatalf("ожидается %s error, получен %s error", expectedErrResponse.Error, expectedErrResponse.Error)
	}
}
