package main

import "github.com/Debianov/calc-ya-go-24/backend/orchestrator"

func StartServer() (err error) {
	s := GetDefaultServer(orchestrator.getHandler())
	err = s.ListenAndServe()
	return
}
