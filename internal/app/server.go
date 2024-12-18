package app

import (
	"github.com/Debianov/calc-ya-go-24/internal"
	"net/http"
)

func StartServer() (err error) {
	var mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", CalcHandler)
	var handler = PanicMiddleware(mux)
	s := internal.GetDefaultServer(handler)
	err = s.ListenAndServe()
	return
}
