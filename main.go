package main

import (
	"fmt"
	"github.com/Debianov/calc-ya-go-24/pkg"
)

func main() {
	//var err error
	//err = agent.StartServer()
	//if err != nil {
	//	panic(err)
	//}
	result, _ := pkg.GeneratePostfix("2 * 3 * 4 + (2 + 3)")
	fmt.Println(result)
}
