package main

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"io"
	"net/http"
)

type Config struct {
}

type JsonPayload interface {
	Marshal() (result []byte, err error)
}

type RequestJson struct {
	Expression string `json:"expression"`
}

func (r RequestJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}

type OKJson struct {
	Result float64 `json:"result"`
}

func (o OKJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&o)
	return
}

type ErrorJson struct {
	Error string `json:"error"`
}

func (e ErrorJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

func expressionValidErrorHandler(w http.ResponseWriter) {
	var (
		buf         []byte
		err         error
		errResponse = &ErrorJson{Error: "Expression is not valid"}
	)
	buf, err = errResponse.Marshal()
	if err != nil {
		panic(err)
	}
	w.WriteHeader(422)
	_, err = w.Write(buf)
	if err != nil {
		panic(err)
	}
	return
}

func calcHandler(w http.ResponseWriter, r *http.Request) {
	var (
		reader        io.ReadCloser
		buf           []byte
		err           error
		requestStruct RequestJson
	)
	if r.Method != http.MethodPost {
		expressionValidErrorHandler(w)
	}
	reader = r.Body
	buf, err = io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(buf, &requestStruct)
	if err != nil {
		panic(err)
	}
	var (
		result float64
	)
	result, err = pkg.Calc(requestStruct.Expression)
	if err != nil {
		expressionValidErrorHandler(w)
	}
	var responseStruct = &OKJson{Result: result}
	buf, err = responseStruct.Marshal()
	if err != nil {
		panic(err)
	}
	_, err = w.Write(buf)
	if err != nil {
		panic(err)
	}
	w.WriteHeader(200)
}

func main() {
	var err error
	http.HandleFunc("/api/v1/calculate", calcHandler)
	s := &http.Server{Addr: "127.0.0.1:8000"}
	err = s.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
