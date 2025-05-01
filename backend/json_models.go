package backend

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"go/types"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

type JsonPayload interface {
	Marshal() (result []byte, err error)
}

type RequestJson struct {
	Expression string `json:"expression"`
}

func (r RequestJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}

/*
RequestNilJson изначально нужен для передачи nil и вызова Internal Server Error. Мы передаём nil, затем
он извлекается через Expression для создания Reader, а этот Reader запихивается в http.Request и передаётся
дальше в функцию. Далее, функция вызовет панику, паника перехватится PanicMiddleware, и далее по списку.

Используется в тесте TestBadGetHandler.
*/
type RequestNilJson struct {
	Expression types.Type `json:"expression"`
}

func (r RequestNilJson) Marshal() (result []byte, err error) {
	return nil, nil
}

type EmptyJson struct {
}

func (e EmptyJson) Marshal() (result []byte, err error) {
	return
}

type ExprStatus string

const (
	Ready        ExprStatus = "Есть готовые задачи"
	NoReadyTasks            = "Нет готовых задач"
	Completed               = "Выполнено"
	Cancelled               = "Отменено"
)

type CommonExpression interface {
	GetId() int
	GetStatus() ExprStatus
	GetReadyGrpcTask() (GrpcTask, error)
	GetResult() int64
	GetTasksHandler() CommonTasksHandler
	UpdateTask(result GrpcResult, timeAt time.Time) (err error)
	JsonPayload
	MarshalId() (result []byte, err error)
	DivideIntoTasks()
}

type Expression struct {
	Id           int          `json:"id"`
	Status       atomic.Value `json:"status"`
	Result       atomic.Int64 `json:"result"`
	postfix      []string
	tasksHandler *TasksHandler
}

func (e *Expression) MarshalJSON() (result []byte, err error) {
	toMarshal := struct {
		Id     int        `json:"id"`
		Status ExprStatus `json:"status"`
		Result int64      `json:"result"`
	}{e.Id, e.GetStatus(), e.GetResult()}
	return json.Marshal(&toMarshal)
}

func (e *Expression) Marshal() (result []byte, err error) {
	result, err = json.Marshal(e)
	return
}

func (e *Expression) MarshalId() (result []byte, err error) {
	result, err = json.Marshal(&struct {
		ID int `json:"id"`
	}{e.Id})
	return
}

func (e *Expression) GetId() int {
	return e.Id
}

// GetStatus потокобезопасен. Не используйте прямой доступ к Status.
func (e *Expression) GetStatus() ExprStatus {
	return e.Status.Load().(ExprStatus)
}

func (e *Expression) GetResult() int64 {
	return e.Result.Load()
}

func (e *Expression) GetReadyGrpcTask() (result GrpcTask, err error) {
	maybeReadyTask := e.tasksHandler.RegisterFirst()
	if maybeReadyTask.IsReadyToCalc() {
		if e.tasksHandler.Len() == 1 {
			e.updateStatus(NoReadyTasks)
		} else {
			e.updateStatus(Ready)
		}
		taskWithTime := e.tasksHandler.sentTasks.WrapWithTime(maybeReadyTask, time.Now())
		taskWithTime.SetStatus(Sent)
		return &taskWithTime, nil
	} else {
		return nil, errors.New("(bug) разработчиком ожидается, что выданный expr (Id %d) " +
			"будет иметь хотя бы 1 готовый к отправке task")
	}
}

func (e *Expression) GetTasksHandler() CommonTasksHandler {
	return e.tasksHandler
}

func (e *Expression) UpdateTask(result GrpcResult, timeAtReceiveTask time.Time) (err error) {
	task, timeAtSendingTask, ok := e.tasksHandler.PopSentTask(int(result.GetPairId()))
	if !ok {
		return &TaskIDNotExist{int(result.GetPairId())}
	}
	if factTime := timeAtReceiveTask.Sub(timeAtSendingTask); factTime > task.GetPermissibleDuration() {
		e.updateStatus(Cancelled)
		return &TimeoutExecution{task.GetPermissibleDuration(), factTime, task.GetOperation(),
			task.GetPairId()}
	}
	task.SetResult(result.GetResult())
	e.tasksHandler.CountUpdatedTask()
	if e.tasksHandler.Len() == 1 {
		e.updateStatus(Completed)
		e.setResult(task.GetResult())
	}
	return
}

func (e *Expression) DivideIntoTasks() {
	var (
		operatorCount int
		stack         = pkg.StackFabric[int64]()
	)
	for _, r := range e.postfix { // TODO: сделать структуру в постфиксе уже распарсеной. нам останется пройтись
		// TODO по ней слева направо и записать всё в порядке <оператор, операнд, операнд>.
		if pkg.IsNumber(r) {
			operandInInt, err := strconv.ParseInt(r, 10, 64)
			if err != nil {
				log.Panic(err)
			}
			stack.Push(operandInInt)
		} else if pkg.IsOperator(r) {
			var (
				newId   = e.generateId(operatorCount)
				newTask InternalTask
			)
			if stack.Len() >= 2 {
				arg2 := stack.Pop()
				arg1 := stack.Pop()
				newTask = CallTaskWithTimeFabric(newId, arg1, arg2, r, e.getPermissibleTime(r), ReadyToCalc)
			} else if stack.Len() == 1 {
				newTask = CallTaskWithTimeFabric(newId, nil, stack.Pop(), r, e.getPermissibleTime(r),
					WaitingOtherTasks)
			} else {
				newTask = CallTaskWithTimeFabric(newId, nil, nil, r, e.getPermissibleTime(r), WaitingOtherTasks)
			}
			e.tasksHandler.Add(newTask)
			operatorCount++
		}
	}
	return
}

func (e *Expression) generateId(operatorCount int) int32 {
	return int32(pkg.Pair(e.Id, operatorCount))
}

func (e *Expression) getPermissibleTime(currentOperator string) (result time.Duration) {
	var (
		operatorAndEnvNamePairs = map[string]EnvVar{"+": *CallEnvVarFabric("TIME_ADDITION", "2s"),
			"-": *CallEnvVarFabric("TIME_SUBTRACTION", "2s"),
			"*": *CallEnvVarFabric("TIME_MULTIPLICATIONS", "2s"),
			"/": *CallEnvVarFabric("TIME_DIVISIONS", "2s")}
		maybeDuration string
		err           error
	)
	for operator, envVar := range operatorAndEnvNamePairs {
		if currentOperator == operator {
			maybeDuration, _ = envVar.Get()
			result, err = time.ParseDuration(maybeDuration)
			if err != nil {
				log.Panic(err)
			}
		}
	}
	return
}

// updateStatus потокобезопасен
func (e *Expression) updateStatus(status ExprStatus) bool {
	return e.Status.CompareAndSwap(e.Status.Load(), status)
}

// setResult потокобезопасен
func (e *Expression) setResult(result int64) bool {
	return e.Result.CompareAndSwap(e.Result.Load(), result)
}

type ExpressionsJsonTitle struct {
	Expressions []CommonExpression `json:"expressions"`
}

func (e *ExpressionsJsonTitle) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

type ExpressionJsonTitle struct {
	Expression CommonExpression `json:"expression"`
}

func (e *ExpressionJsonTitle) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

func CallExpressionFabric(postfix []string, Id int, status ExprStatus, tasksHandler *TasksHandler) (newInstance *Expression) {
	newInstance = &Expression{postfix: postfix, Id: Id, tasksHandler: tasksHandler}
	newInstance.Status.Swap(status)
	return
}

type TaskStatus int

const (
	ReadyToCalc TaskStatus = iota
	Sent
	WaitingOtherTasks
	Calculated
)

type AgentResult struct {
	Id     int   `json:"id"`
	Result int64 `json:"result"`
}

func (a *AgentResult) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&a)
	return
}

/*
InternalTask реализует Task-и, которые обращаются исключительно внутри оркестратора и не используются
для передачи через GRPC.
*/
type InternalTask interface {
	CommonTask
	GetArg1() (int64, bool)
	GetArg2() (int64, bool)
	GetOperation() string
	ResultHolder
	GetStatus() TaskStatus
	GetPermissibleDuration() time.Duration
	IsReadyToCalc() bool
	SetStatus(newStatus TaskStatus)
	SetArg1(int64)
	SetArg2(int64)
	SetResult(result int64) bool
}

type Task struct {
	pairId          int32
	arg1            interface{}
	arg2            interface{}
	operation       string
	permissibleTime time.Duration
	status          atomic.Value
	result          atomic.Int64
}

func (t *Task) GetPairId() int32 {
	return t.pairId
}

func (t *Task) GetOperation() string {
	return t.operation
}

// GetStatus потокобезопасен.
func (t *Task) GetStatus() TaskStatus {
	return t.status.Load().(TaskStatus)
}

// GetResult потокобезопасен.
func (t *Task) GetResult() int64 {
	return t.result.Load()
}

// SetStatus потокобезопасен.
func (t *Task) SetStatus(newStatus TaskStatus) {
	t.status.CompareAndSwap(t.status.Load(), newStatus)
}

func (t *Task) IsReadyToCalc() bool {
	return t.status.Load().(TaskStatus) == ReadyToCalc
}

func (t *Task) GetArg1() (int64, bool) {
	v, ok := t.arg1.(int64)
	return v, ok
}

func (t *Task) GetArg2() (int64, bool) {
	v, ok := t.arg2.(int64)
	return v, ok
}

func (t *Task) SetArg1(result int64) {
	t.arg1 = result
}

func (t *Task) SetArg2(result int64) {
	t.arg2 = result
}

// SetResult потокобезопасен.
func (t *Task) SetResult(result int64) bool {
	return t.result.CompareAndSwap(t.result.Load(), result)
}

func (t *Task) GetPermissibleDuration() time.Duration {
	return t.permissibleTime
}

/*
CallTaskFabric arg1 и arg2 должны быть либо nil, либо int(8/32/64)
*/
func CallTaskFabric(pairId int32, arg1 interface{}, arg2 interface{}, operation string,
	status TaskStatus) (newInstance *Task) {
	var (
		finalArg1 interface{}
		finalArg2 interface{}
		err       error
	)
	finalArg1, err = convertToInt64Interface(arg1)
	if err != nil {
		panic(err)
	}
	finalArg2, err = convertToInt64Interface(arg2)
	if err != nil {
		panic(err)
	}
	newInstance = &Task{
		pairId:    pairId,
		arg1:      finalArg1,
		arg2:      finalArg2,
		operation: operation,
	}
	newInstance.SetStatus(status)
	return newInstance
}

/*
CallTaskWithTimeFabric arg1 и arg2 должны быть либо nil, либо int(8/32/64)
*/
func CallTaskWithTimeFabric(pairId int32, arg1 interface{}, arg2 interface{}, operation string,
	permissibleTime time.Duration, status TaskStatus) (newInstance *Task) {
	var (
		finalArg1 interface{}
		finalArg2 interface{}
		err       error
	)
	finalArg1, err = convertToInt64Interface(arg1)
	if err != nil {
		panic(err)
	}
	finalArg2, err = convertToInt64Interface(arg2)
	if err != nil {
		panic(err)
	}
	newInstance = &Task{
		pairId:          pairId,
		arg1:            finalArg1,
		arg2:            finalArg2,
		operation:       operation,
		permissibleTime: permissibleTime,
	}
	newInstance.SetStatus(status)
	return newInstance
}
