package backend

import (
	"time"
)

type CommonTask interface {
	GetPairId() int32
	GetOperation() string
}

type InternalTask interface {
	CommonTask
	GetArg1() (int64, bool)
	GetArg2() (int64, bool)
	SetArg1(int64)
	SetArg2(int64)
	GetResult() int64
	WriteResult(result int64) error
	SetStatus(newStatus TaskStatus)
	IsReadyToCalc() bool
	GetOperationTime() time.Duration
}

type GrpcTask interface {
	CommonTask
	GetArg1() int64
	GetArg2() int64
	GetOperationDuration() string
}

type TaskWithTime struct {
	task              *Task
	timeAtSendingTask time.Time
}

func (t *TaskWithTime) GetPairId() int32 {
	return t.task.GetPairId()
}

func (t *TaskWithTime) GetOperation() string {
	return t.task.GetOperation()
}

func (t *TaskWithTime) GetOperationDuration() string {
	return t.task.GetOperationTime().String()
}

func (t *TaskWithTime) GetArg1() int64 {
	return t.task.Arg1.(int64)
}

func (t *TaskWithTime) GetArg2() int64 {
	return t.task.Arg2.(int64)
}

func (t *TaskWithTime) GetResult() int64 {
	return t.task.GetResult()
}

func (t *TaskWithTime) WriteResult(result int64) error {
	return t.task.WriteResult(result)
}

func (t *TaskWithTime) SetStatus(newStatus TaskStatus) {
	t.task.SetStatus(newStatus)
}

func (t *TaskWithTime) IsReadyToCalc() bool {
	return t.task.IsReadyToCalc()
}
