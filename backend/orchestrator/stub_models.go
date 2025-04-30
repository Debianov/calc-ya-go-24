package main

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	"time"
)

type StubExpressionsList struct {
	buf    []ExpressionStub
	cursor int
}

func (s *StubExpressionsList) AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int) {
	//TODO implement me
	panic("implement me")
}

func (s *StubExpressionsList) GetAllExprs() (result []backend.CommonExpression) {
	for _, expr := range s.buf {
		result = append(result, backend.CommonExpression(&expr))
	}
	return
}

func (s *StubExpressionsList) Get(id int) (result backend.CommonExpression, ok bool) {
	if id < len(s.buf) {
		ok = true
		result = backend.CommonExpression(&s.buf[id])
	} else {
		ok = false
	}
	return
}

func (s *StubExpressionsList) GetReadyExpr() (result backend.CommonExpression) {
	var expr ExpressionStub
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
func callStubExprsListFabric(expressions ...ExpressionStub) (result *StubExpressionsList) {
	if len(expressions) == 0 {
		result = &StubExpressionsList{}
	} else {
		result = &StubExpressionsList{buf: expressions}
	}
	return
}

type ExpressionStub struct {
	Id           int                `json:"id"`
	Status       backend.ExprStatus `json:"status"`
	Result       int64              `json:"result"`
	TasksHandler StubTasksHandler
}

func (s *ExpressionStub) Marshal() (result []byte, err error) {
	return json.Marshal(s)
}

func (s *ExpressionStub) MarshalId() (result []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionStub) GetId() int {
	return s.Id
}

func (s *ExpressionStub) GetStatus() backend.ExprStatus {
	return s.Status
}

func (s *ExpressionStub) GetResult() int64 {
	panic("implement me")
}

func (s *ExpressionStub) GetReadyGrpcTask() (backend.GrpcTask, error) {
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

func (s *ExpressionStub) GetTasksHandler() backend.CommonTasksHandler {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionStub) UpdateTask(taskID int, result int64, timeAtReceiveTask time.Time) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionStub) DivideIntoTasks() {
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
	v, _ := s.Task.GetArg1()
	return v
}

func (s *StubTaskWithTime) GetArg2() int64 {
	v, _ := s.Task.GetArg2()
	return v
}

func (s *StubTaskWithTime) GetPermissibleDuration() string {
	return s.Task.GetPermissibleDuration().String()
}

type ExpressionJsonStub struct {
	ID int `json:"id"`
}

func (e ExpressionJsonStub) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

type ExpressionsJsonTitleStub struct {
	Expressions []ExpressionStub `json:"expressions"`
}

func (e *ExpressionsJsonTitleStub) Marshal() (result []byte, err error) {
	result, err = json.Marshal(e)
	return
}

type ExpressionJsonTitleStub struct {
	Expression ExpressionStub `json:"expression"`
}

func (e *ExpressionJsonTitleStub) Marshal() (result []byte, err error) {
	result, err = json.Marshal(e)
	return
}
