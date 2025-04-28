package backend

import (
	"time"
)

type CommonTask interface {
	GetPairId() int32
	GetOperation() string
	GetStatus() TaskStatus
	GetResult() int64
	SetStatus(newStatus TaskStatus)
	IsReadyToCalc() bool
}

type GrpcTask interface {
	CommonTask
	GetArg1() int64
	GetArg2() int64
	GetPermissibleDuration() string
	GetWrappedTask() InternalTask
	GetTimeAtSendingTask() time.Time
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

func (t *TaskWithTime) GetStatus() TaskStatus {
	return t.task.GetStatus()
}

func (t *TaskWithTime) GetResult() int64 {
	return t.task.GetResult()
}

func (t *TaskWithTime) SetStatus(newStatus TaskStatus) {
	t.task.SetStatus(newStatus)
}

func (t *TaskWithTime) IsReadyToCalc() bool {
	return t.task.IsReadyToCalc()
}

func (t *TaskWithTime) GetArg1() int64 {
	return t.task.Arg1.(int64)
}

func (t *TaskWithTime) GetArg2() int64 {
	return t.task.Arg2.(int64)
}

func (t *TaskWithTime) GetPermissibleDuration() string {
	return t.task.GetPermissibleDuration().String()
}

func (t *TaskWithTime) GetWrappedTask() InternalTask {
	return t.task
}

func (t *TaskWithTime) GetTimeAtSendingTask() time.Time {
	return t.timeAtSendingTask
}
