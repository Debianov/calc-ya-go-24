package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Debianov/calc-ya-go-24/backend"
	"time"
)

type ExpressionsListStub struct {
	buf    []ExpressionStub
	cursor int
}

func (s *ExpressionsListStub) AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionsListStub) GetAllExprs() (result []backend.CommonExpression) {
	for _, expr := range s.buf {
		result = append(result, backend.CommonExpression(&expr))
	}
	return
}

// Get получает ExpressionStub из buf по порядковому id Expression в этом buf.
func (s *ExpressionsListStub) Get(id int) (result backend.CommonExpression, ok bool) {
	if id < len(s.buf) {
		ok = true
		result = backend.CommonExpression(&s.buf[id])
	} else {
		ok = false
	}
	return
}

func (s *ExpressionsListStub) GetReadyExpr() (result backend.CommonExpression) {
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
callExprsListStubFabric формирует новый ExpressionsListStub, который может быть присвоен глобальной
переменной exprsList для подмены настоящего списка в целях тестирования, или использоваться как-то ещё.
*/
func callExprsListStubFabric(expressions ...ExpressionStub) (result *ExpressionsListStub) {
	if len(expressions) == 0 {
		result = &ExpressionsListStub{}
	} else {
		result = &ExpressionsListStub{buf: expressions}
	}
	return
}

type ExpressionStub struct {
	Id           int                `json:"id"`
	Status       backend.ExprStatus `json:"status"`
	Result       int64              `json:"result"`
	TasksHandler *TasksHandlerStub
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
		newTask TaskWithTimeStub
	)
	for _, task := range s.TasksHandler.Buf {
		if task.IsReadyToCalc() {
			newTask = TaskWithTimeStub{
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

func (s *ExpressionStub) UpdateTask(result backend.GrpcResult, _ time.Time) (err error) {
	taskToChange, ok := s.TasksHandler.Buf[result.GetPairId()]
	if !ok {
		return fmt.Errorf("задачи %d не найдено", result.GetPairId())
	}
	taskToChange.SetResult(result.GetResult())
	return
}

func (s *ExpressionStub) DivideIntoTasks() {
	//TODO implement me
	panic("implement me")
}

type TasksHandlerStub struct {
	Buf map[int32]backend.InternalTask
}

func (s *TasksHandlerStub) Add(task backend.InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) Get(ind int) backend.InternalTask {
	return s.Buf[int32(ind)]
}

func (s *TasksHandlerStub) Len() int {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) RegisterFirst() (task backend.InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) CountUpdatedTask() {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) PopSentTask(taskId int) (backend.InternalTask, time.Time, bool) {
	//TODO implement me
	panic("implement me")
}

type TaskWithTimeStub struct {
	Task      *backend.Task
	DummyTime time.Time
}

func (s *TaskWithTimeStub) GetPairId() int32 {
	return s.Task.GetPairId()
}

func (s *TaskWithTimeStub) GetOperation() string {
	return s.Task.GetOperation()
}

func (s *TaskWithTimeStub) GetArg1() int64 {
	v, _ := s.Task.GetArg1()
	return v
}

func (s *TaskWithTimeStub) GetArg2() int64 {
	v, _ := s.Task.GetArg2()
	return v
}

func (s *TaskWithTimeStub) GetPermissibleDuration() string {
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
