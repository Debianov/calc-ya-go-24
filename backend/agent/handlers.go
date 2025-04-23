package main

import (
	"errors"
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
)

func Calc(task *pb.TaskToSend) (agentResult *pb.TaskResult, err error) {
	var result int64
	agentResult = &pb.TaskResult{
		PairId: task.PairId,
	}
	switch task.Operation {
	case "+":
		result = task.Arg1 + task.Arg2
	case "-":
		result = task.Arg1 - task.Arg2
	case "*":
		result = task.Arg1 * task.Arg2
	case "/":
		result = task.Arg1 / task.Arg2
	default:
		err = errors.New("неизвестная операция")
		return
	}
	agentResult.Result = result
	return
}
