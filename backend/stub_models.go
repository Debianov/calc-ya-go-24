package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ExpressionStub struct {
	Id           int        `json:"id"`
	Status       ExprStatus `json:"status"`
	Result       int64      `json:"result"`
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

func (s *ExpressionStub) GetStatus() ExprStatus {
	return s.Status
}

func (s *ExpressionStub) GetResult() int64 {
	panic("implement me")
}

func (s *ExpressionStub) GetOwnerId() int64 {
	panic("implement me")
}

func (s *ExpressionStub) GetReadyGrpcTask() (GrpcTask, error) {
	var (
		newTask TaskWithTimeStub
	)
	for _, task := range s.TasksHandler.Buf {
		if task.IsReadyToCalc() {
			newTask = TaskWithTimeStub{
				Task:      task.(*Task),
				DummyTime: time.Now(),
			}
			return GrpcTask(&newTask), nil
		}
	}
	return nil, errors.New("no ready tasks")
}

func (s *ExpressionStub) GetTasksHandler() CommonTasksHandler {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionStub) UpdateTask(result GrpcResult, _ time.Time) (err error) {
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
	Buf map[int32]InternalTask
}

func (s *TasksHandlerStub) Add(task InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) Get(ind int) InternalTask {
	return s.Buf[int32(ind)]
}

func (s *TasksHandlerStub) Len() int {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) RegisterFirst() (task InternalTask) {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) CountUpdatedTask() {
	//TODO implement me
	panic("implement me")
}

func (s *TasksHandlerStub) PopSentTask(taskId int) (InternalTask, time.Time, bool) {
	//TODO implement me
	panic("implement me")
}

type TaskWithTimeStub struct {
	Task      *Task
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

type RequestJsonStub struct {
	Token      string `json:"token"`
	Expression string `json:"expression"`
}

func (r *RequestJsonStub) Marshal() (result []byte, err error) {
	return json.Marshal(r)
}

type UserStub struct {
	hashMan        HashMan
	Login          string `json:"login"`
	Password       string `json:"password"`
	Id             int64
	hashedPassword string
}

func (u *UserStub) Marshal() (result []byte, err error) {
	return json.Marshal(u)
}

func (u *UserStub) GetLogin() string {
	return u.Login
}

func (u *UserStub) SetLogin(s string) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetId() int64 {
	return u.Id
}

func (u *UserStub) SetId(i int64) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetPassword() string {
	return u.Password
}

func (u *UserStub) SetPassword(password string) {
	u.Password = password
}

func (u *UserStub) GetHashedPassword() string {
	return u.hashedPassword
}

func (u *UserStub) SetHashedPassword(salt string) (err error) {
	u.hashedPassword, err = u.hashMan.Generate(salt)
	return
}

func (u *UserStub) Is(user UserWithPassword) (status bool) {
	var (
		err error
	)
	if err = u.hashMan.Compare(u.GetHashedPassword(), user.GetPassword()); err != nil {
		return
	}
	status = true
	return
}

type JwtTokenJsonWrapperStub struct {
	Token string `json:"token"`
}

func (j *JwtTokenJsonWrapperStub) Marshal() (result []byte, err error) {
	return json.Marshal(j)
}
