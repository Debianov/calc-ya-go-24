package main

import (
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
)

func Calc(task *pb.TaskToSend) (agentResult *pb.TaskResult, err error) {
	var result int64
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
		err = unknownOperator
		return
	}
	agentResult = &pb.TaskResult{
		PairId: task.PairId,
		Result: result,
	}
	return
}
