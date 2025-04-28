package main

import (
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	"time"
)

type StubExpressionsList struct {
	buf    []StubExpression
	cursor int
}

func (s *StubExpressionsList) AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpressionsList) GetAllExprs() []backend.CommonExpression {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpressionsList) Get(id int) (backend.CommonExpression, bool) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpressionsList) GetReadyExpr() (result backend.CommonExpression) {
	var expr StubExpression
	for _, expr = range s.buf {
		if expr.GetStatus() == backend.Ready {
			result = &expr
			return
		}
	}
	return nil
}

/*
callStubExprsListFabric формирует новый StubExpressionsList, который может быть присвоен глобальной
переменной exprsList для подмены настоящего списка в целях тестирования, или использоваться как-то ещё.
*/
func callStubExprsListFabric(expressions ...StubExpression) (result *StubExpressionsList) {
	if len(expressions) == 0 {
		result = &StubExpressionsList{}
	} else {
		result = &StubExpressionsList{buf: expressions}
	}
	return
}

type StubExpression struct {
	ID           int
	Status       backend.ExprStatus
	TasksHandler StubTasksHandler
}

func (s *StubExpression) Marshal() (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpression) MarshalId() (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpression) GetId() int {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpression) GetStatus() backend.ExprStatus {
	return s.Status
}

func (s *StubExpression) GetReadyGrpcTask() (backend.GrpcTask, error) {
	var (
		newTask StubTaskWithTime
	)
	for _, task := range s.TasksHandler.Buf {
		if task.IsReadyToCalc() {
			newTask = StubTaskWithTime{
				Task:      task.(*backend.Task),
				DummyTime: time.Now(),
			}
			return backend.GrpcTask(&newTask), nil
		}
	}
	return nil, errors.New("no ready tasks")
}

func (s *StubExpression) GetTasksHandler() backend.CommonTasksHandler {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpression) UpdateTask(taskID int, result int64, timeAtReceiveTask time.Time) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpression) DivideIntoTasks() {
	//TODO implement me
	panic("implement me")
}

type StubTasksHandler struct {
	Buf map[int]backend.InternalTask
}

func (s *StubTasksHandler) Add(task backend.InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *StubTasksHandler) Get(ind int) backend.InternalTask {
	//TODO implement me
	panic("implement me")
}

func (s *StubTasksHandler) Len() int {
	//TODO implement me
	panic("implement me")
}

func (s *StubTasksHandler) RegisterFirst() (task backend.InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *StubTasksHandler) CountUpdatedTask() {
	//TODO implement me
	panic("implement me")
}

func (s *StubTasksHandler) PopSentTask(taskId int) (backend.InternalTask, time.Time, bool) {
	//TODO implement me
	panic("implement me")
}

type StubTaskWithTime struct {
	Task      *backend.Task
	DummyTime time.Time
}

func (s *StubTaskWithTime) GetPairId() int32 {
	return s.Task.GetPairId()
}

func (s *StubTaskWithTime) GetOperation() string {
	return s.Task.GetOperation()
}

func (s *StubTaskWithTime) GetArg1() int64 {
	return s.Task.Arg1.(int64)
}

func (s *StubTaskWithTime) GetArg2() int64 {
	return s.Task.Arg2.(int64)
}

func (s *StubTaskWithTime) GetPermissibleDuration() string {
	return s.Task.GetPermissibleDuration().String()
}
