package main

import "github.com/Debianov/calc-ya-go-24/backend/agent"

func main() {
	var err error
	err = agent.StartServer()
	if err != nil {
		panic(err)
	}
}
