package main

import "github.com/Debianov/calc-ya-go-24/internal/app"

func main() {
	var err error
	err = app.StartServer()
	if err != nil {
		panic(err)
	}
}
