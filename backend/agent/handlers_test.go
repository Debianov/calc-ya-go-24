package main

import (
	pb "github.com/Debianov/calc-ya-go-24/backend/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testCalcUnknownOperatorErr(t *testing.T) {
	var (
		agentResult   *pb.TaskResult
		err           error
		toSendStructs = []*pb.TaskToSend{
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "-/",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "]",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "\v",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "឵\x20\x00",
				PermissibleDuration: "",
			},
		}
	)
	for ind, toSend := range toSendStructs {
		agentResult, err = Calc(toSend)
		assert.Equal(t, (*pb.TaskResult)(nil), agentResult, "case %d", ind)
		assert.ErrorIs(t, unknownOperator, err, "case %d", ind)
	}
}

func testCalcOk(t *testing.T) {
	var (
		agentResult   *pb.TaskResult
		err           error
		toSendStructs = []*pb.TaskToSend{
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "-",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                4,
				Arg2:                2,
				Operation:           "+",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                2,
				Arg2:                3,
				Operation:           "-",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                3,
				Arg2:                2,
				Operation:           "/",
				PermissibleDuration: "",
			},
			{
				PairId:              0,
				Arg1:                100,
				Arg2:                2,
				Operation:           "*",
				PermissibleDuration: "",
			},
		}
		expectedStructs = []*pb.TaskResult{
			{
				PairId: 0,
				Result: 2,
			},
			{
				PairId: 0,
				Result: 6,
			},
			{
				PairId: 0,
				Result: -1,
			},
			{
				PairId: 0,
				Result: 1,
			},
			{
				PairId: 0,
				Result: 200,
			},
		}
	)
	for ind, toSend := range toSendStructs {
		agentResult, err = Calc(toSend)
		assert.Equal(t, expectedStructs[ind], agentResult, "case %d", ind)
		assert.ErrorIs(t, nil, err, "case %d", ind)
	}
}

func TestCalc(t *testing.T) {
	t.Run("UnknowOperatorErr", testCalcUnknownOperatorErr)
	t.Run("Ok", testCalcOk)
}
