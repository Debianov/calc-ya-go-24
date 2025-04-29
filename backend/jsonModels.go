package backend

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"go/types"
	"log"
	"strconv"
	"sync"
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

type OKJson struct {
	Result float64 `json:"result"`
}

func (o OKJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&o)
	return
}

type ErrorJson struct {
	Error string `json:"error"`
}

func (e ErrorJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
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
	JsonPayload
	MarshalId() (result []byte, err error)
	GetId() int
	GetStatus() ExprStatus
	GetReadyGrpcTask() (GrpcTask, error)
	GetTasksHandler() CommonTasksHandler
	UpdateTask(taskID int, result int64, timeAtReceiveTask time.Time) (err error)
	DivideIntoTasks()
}

type Expression struct {
	postfix      []string
	ID           int        `json:"id"`
	Status       ExprStatus `json:"status"`
	Result       int64      `json:"result"`
	tasksHandler *TasksHandler
	mut          sync.Mutex
}

func (e *Expression) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

func (e *Expression) MarshalId() (result []byte, err error) {
	result, err = json.Marshal(&struct {
		ID int `json:"id"`
	}{e.ID})
	return
}

func (e *Expression) GetId() int {
	return e.ID
}

func (e *Expression) GetStatus() ExprStatus {
	return e.Status
}

func (e *Expression) GetReadyGrpcTask() (result GrpcTask, err error) {
	maybeReadyTask := e.tasksHandler.RegisterFirst()
	if maybeReadyTask.IsReadyToCalc() {
		if e.tasksHandler.Len() == 1 {
			e.changeStatus(NoReadyTasks)
		} else {
			e.changeStatus(Ready)
		}
		taskWithTime := e.tasksHandler.sentTasks.WrapWithTime(maybeReadyTask, time.Now())
		taskWithTime.SetStatus(Sent)
		return &taskWithTime, nil
	} else {
		return nil, errors.New("(bug) разработчиком ожидается, что выданный expr (id %d) " +
			"будет иметь хотя бы 1 готовый к отправке task")
	}
}

func (e *Expression) GetTasksHandler() CommonTasksHandler {
	return e.tasksHandler
}

func (e *Expression) UpdateTask(taskID int, result int64, timeAtReceiveTask time.Time) (err error) {
	task, timeAtSendingTask, ok := e.tasksHandler.PopSentTask(taskID)
	if !ok {
		return &TaskIDNotExist{taskID}
	}
	if factTime := timeAtReceiveTask.Sub(timeAtSendingTask); factTime > task.GetPermissibleDuration() {
		e.changeStatus(Cancelled)
		return &TimeoutExecution{task.GetPermissibleDuration(), factTime, task.GetOperation(),
			task.GetPairId()}
	}
	task.SetResult(result)
	// UpdateExpression
	e.tasksHandler.CountUpdatedTask()
	if e.tasksHandler.Len() == 1 {
		e.changeStatus(Completed)
		e.writeResult(task.GetResult())
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
				newTask = CallTaskFabricWithTime(newId, arg1, arg2, r, e.getPermissibleTime(r), ReadyToCalc)
			} else if stack.Len() == 1 {
				newTask = CallTaskFabricWithTime(newId, nil, stack.Pop(), r, e.getPermissibleTime(r),
					WaitingOtherTasks)
			} else {
				newTask = CallTaskFabricWithTime(newId, nil, nil, r, e.getPermissibleTime(r), WaitingOtherTasks)
			}
			e.tasksHandler.Add(newTask)
			operatorCount++
		}
	}
	return
}

func (e *Expression) generateId(operatorCount int) int32 {
	return int32(pkg.Pair(e.ID, operatorCount))
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

func (e *Expression) changeStatus(status ExprStatus) {
	e.mut.Lock()
	defer e.mut.Unlock()
	if e.Status == status {
		return
	}
	if e.Status != Completed && e.Status != Cancelled {
		e.Status = status
	} else {
		log.Printf("попытка изменения статуса выражения %d, когда его статус %v", e.ID, e.Status)
	}
}

func (e *Expression) writeResult(result int64) {
	e.mut.Lock()
	defer e.mut.Unlock()
	e.Result = result
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

type TaskStatus int

const (
	ReadyToCalc TaskStatus = iota
	Sent
	WaitingOtherTasks
	Calculated
)

type AgentResult struct {
	ID     int   `json:"ID"`
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
	GetResult() int64
	GetStatus() TaskStatus
	SetStatus(newStatus TaskStatus)
	IsReadyToCalc() bool
	GetPermissibleDuration() time.Duration
	SetArg1(int64)
	SetArg2(int64)
	SetResult(result int64) bool
}

type Task struct {
	pairID          int32
	arg1            interface{}
	arg2            interface{}
	operation       string
	permissibleTime time.Duration
	status          atomic.Value
	result          atomic.Int64
	mut             sync.Mutex
}

func (t *Task) GetPairId() int32 {
	return t.pairID
}

func (t *Task) GetOperation() string {
	return t.operation
}

func (t *Task) GetStatus() TaskStatus {
	return t.status.Load().(TaskStatus)
}

func (t *Task) GetResult() int64 {
	return t.result.Load()
}

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

func (t *Task) SetResult(result int64) bool {
	return t.result.CompareAndSwap(t.result.Load(), result)
}

func (t *Task) GetPermissibleDuration() time.Duration {
	return t.permissibleTime
}

func CallTaskFabric(pairId int32, arg1 interface{}, arg2 interface{}, operation string,
	status TaskStatus) (newInstance *Task) {
	newInstance = &Task{
		pairID:    pairId,
		arg1:      arg1,
		arg2:      arg2,
		operation: operation,
	}
	newInstance.SetStatus(status)
	return newInstance
}

func CallTaskFabricWithTime(pairId int32, arg1 interface{}, arg2 interface{}, operation string, permissibleTime time.Duration,
	status TaskStatus) (newInstance *Task) {
	newInstance = &Task{
		pairID:          pairId,
		arg1:            arg1,
		arg2:            arg2,
		operation:       operation,
		permissibleTime: permissibleTime,
	}
	newInstance.SetStatus(status)
	return newInstance
}
