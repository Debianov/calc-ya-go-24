package app

import (
	"github.com/Debianov/calc-ya-go-24/internal"
)

func StartServer() (err error) {
	s := internal.GetDefaultServer(getHandler())
	err = s.ListenAndServe()
	return
}
