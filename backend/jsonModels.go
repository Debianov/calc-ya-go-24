package backend

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"go/types"
	"log"
	"strconv"
	"sync"
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
	err = task.WriteResult(result)
	if err != nil {
		log.Panic(err)
	}
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
				newTask = &Task{PairID: newId, Arg2: stack.Pop(), Arg1: stack.Pop(),
					Operation: r, PermissibleTime: e.getOperationTime(r), Status: ReadyToCalc}
			} else if stack.Len() == 1 {
				newTask = &Task{PairID: newId, Arg2: stack.Pop(), Operation: r,
					PermissibleTime: e.getOperationTime(r), Status: WaitingOtherTasks}
			} else {
				newTask = &Task{PairID: newId, Operation: r, PermissibleTime: e.getOperationTime(r),
					Status: WaitingOtherTasks}
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

func (e *Expression) getOperationTime(currentOperator string) (result time.Duration) {
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

// InternalTask реализует Task, которые обращаются исключительно внутри оркестратора.
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
	WriteResult(result int64) error
}

type Task struct {
	PairID          int32         `json:"id"`
	Arg1            interface{}   `json:"arg1"`
	Arg2            interface{}   `json:"arg2"`
	Operation       string        `json:"operation"`
	PermissibleTime time.Duration `json:"operationTime"`
	result          int64
	Status          TaskStatus
	mut             sync.Mutex
}

func (t *Task) GetPairId() int32 {
	return t.PairID
}

func (t *Task) GetOperation() string {
	return t.Operation
}

func (t *Task) GetStatus() TaskStatus {
	return t.Status
}

func (t *Task) GetResult() int64 {
	return t.result
}

func (t *Task) SetStatus(newStatus TaskStatus) {
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.Status == newStatus {
		return
	}
	if t.Status != Calculated && t.Status != newStatus {
		t.Status = newStatus
	}
}

func (t *Task) IsReadyToCalc() bool {
	return t.Status == ReadyToCalc
}

func (t *Task) GetArg1() (int64, bool) {
	v, ok := t.Arg1.(int64)
	return v, ok
}

func (t *Task) GetArg2() (int64, bool) {
	v, ok := t.Arg2.(int64)
	return v, ok
}

func (t *Task) SetArg1(result int64) {
	t.Arg1 = result
}

func (t *Task) SetArg2(result int64) {
	t.Arg2 = result
}

func (t *Task) WriteResult(result int64) error {
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.Status == Sent {
		t.result = result
		t.Status = Calculated
	} else if t.Status == Calculated {
		return errors.New("BUG: разработчиком ожидается, что результат одной и той же задачи не может быть записан" +
			" больше одного раза")
	}
	return nil
}

func (t *Task) GetPermissibleDuration() time.Duration {
	return t.PermissibleTime
}
