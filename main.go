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
	result, _ := pkg.GeneratePostfix("2+2*4")
	b, _ := pkg.EvaluatePostfix(result)
	fmt.Println(b, result)
	//fmt.Println(pkg.Pair(1, 2))
}
