package app

import (
	"github.com/Debianov/calc-ya-go-24/internal"
	"net/http"
)

func StartServer() (err error) {
	http.HandleFunc("/api/v1/calculate", CalcHandler)
	s := internal.GetDefaultServer()
	err = s.ListenAndServe()
	return
}
