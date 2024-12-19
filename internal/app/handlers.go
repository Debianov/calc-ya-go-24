package app

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"io"
	"net/http"
)

func CalcHandler(w http.ResponseWriter, r *http.Request) {
	var (
		reader        io.ReadCloser
		buf           []byte
		err           error
		requestStruct RequestJson
	)
	if r.Method != http.MethodPost {
		return
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
		return
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

func PanicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := recover(); err != nil {
			internalServerErrorHandler(w)
			panic(err)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func internalServerErrorHandler(w http.ResponseWriter) {
	var (
		buf         []byte
		err         error
		errResponse = &ErrorJson{Error: "Internal server error"}
	)
	buf, err = errResponse.Marshal()
	if err != nil {
		panic(err)
	}
	w.WriteHeader(500)
	_, err = w.Write(buf)
	if err != nil {
		panic(err)
	}
	return
}

func getHandler() (handler http.Handler) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", CalcHandler)
	handler = PanicMiddleware(mux)
	return
}
