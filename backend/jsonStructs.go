package backend

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"go/types"
	"log"
	"os"
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

type ExprStatus int

const (
	Ready ExprStatus = iota
	NoReadyTasks
	Completed
	Cancelled
)

const (
	TIME_ADDITION_MS        string = "TIME_ADDITION_MS"
	TIME_SUBTRACTION_MS            = "TIME_SUBTRACTION_MS"
	TIME_MULTIPLICATIONS_MS        = "TIME_MULTIPLICATIONS_MS"
	TIME_DIVISIONS_MS              = "TIME_DIVISIONS_MS"
)

type TaskToSend struct {
	Task              *Task `json:"task"`
	timeAtSendingTask time.Time
}

type Expression struct {
	Postfix      []string
	ID           int        `json:"id"`
	Status       ExprStatus `json:"status"`
	Result       int        `json:"result"`
	TasksHandler *Tasks
	mut          sync.Mutex
}

func (e *Expression) DivideIntoTasks() {
	var (
		operand               string
		operandsBeforeOperand []int64
		operatorCount         int
	)
	for _, r := range e.Postfix { // TODO: сделать структуру в постфиксе, уже распарсенную. нам останется пройтись
		// TODO по ней слева направо и записать всё в порядке <оператор, операнд, операнд>.
		if r == " " {
			if operand != "" {
				operandInInt, err := strconv.ParseInt(operand, 10, 64)
				if err != nil {
					log.Panic()
				}
				operandsBeforeOperand = append(operandsBeforeOperand, operandInInt)
				operand = ""
			} else {
				continue
			}
		} else if pkg.IsNumber(r) {
			operand += r
		} else if pkg.IsOperator(r) {
			var (
				newId   = e.generateId(operatorCount)
				newTask *Task
			)
			if len(operandsBeforeOperand) == 2 {
				newTask = &Task{PairID: newId, Arg1: operandsBeforeOperand[0], Arg2: operandsBeforeOperand[1],
					Operation: r, OperationTime: e.getOperationTime(r), status: ReadyToCalc}
			} else if len(operandsBeforeOperand) == 1 {
				newTask = &Task{PairID: newId, Arg2: operandsBeforeOperand[0], Operation: r,
					OperationTime: e.getOperationTime(r), status: WaitingOtherTasks}
			} else {
				newTask = &Task{PairID: newId, Operation: r, OperationTime: e.getOperationTime(r), status: WaitingOtherTasks}
			}
			e.TasksHandler.add(newTask)
			operandsBeforeOperand = make([]int64, 0)
			operatorCount++
		}
	}
	return
}

func (e *Expression) generateId(operatorCount int) int {
	return pkg.Pair(e.ID, operatorCount)
}

func (e *Expression) getOperationTime(currentOperator string) (result time.Duration) {
	var (
		operatorAndEnvNamePairs = map[string]string{"+": TIME_ADDITION_MS, "-": TIME_SUBTRACTION_MS,
			"*": TIME_MULTIPLICATIONS_MS, "/": TIME_DIVISIONS_MS}
		maybeDuration string
		err           error
	)
	for operator, envName := range operatorAndEnvNamePairs {
		if currentOperator == operator {
			maybeDuration = os.Getenv(envName)
			if maybeDuration == "" {
				log.Printf("WARNING: переменная %s не обнаружена", envName)
			}
			result, err = time.ParseDuration(maybeDuration)
			if err != nil {
				log.Panic(err)
			}
		}
	}
	return
}

func (e *Expression) GetReadyToSendTask() TaskToSend {
	maybeReadyTask := e.TasksHandler.getFirst()
	if maybeReadyTask.IsReadyToCalc() {
		e.changeStatus(Ready)
		taskToSend := e.TasksHandler.fabricAppendInSentTasks(maybeReadyTask, time.Now())
		return taskToSend
	} else {
		e.changeStatus(NoReadyTasks)
		return TaskToSend{}
	}
}

func (e *Expression) changeStatus(status ExprStatus) {
	e.mut.Lock()
	defer e.mut.Unlock()
	if e.Status != status {
		return
	}
	if e.Status != Completed && e.Status != Cancelled {
		e.Status = status
	} else {
		log.Printf("попытка изменения статуса выражения %d, когда его статус %v", e.ID, e.Status)
	}
}

func (e *Expression) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

func (e *Expression) MarshalID() (result []byte, err error) {
	result, err = json.Marshal(&Expression{ID: e.ID})
	return
}

func (e *Expression) WriteResultIntoTask(taskID int, result int, timeAtReceiveTask time.Time) (err error) {
	task, timeAtSendingTask, ok := e.TasksHandler.getTask(taskID)
	if factTime := timeAtReceiveTask.Sub(timeAtSendingTask); factTime > task.OperationTime {
		e.changeStatus(Cancelled)
		return TimeoutExecution{task.OperationTime, factTime, task.Operation,
			task.PairID}
	}
	if !ok {
		return TaskIDNotExist{taskID}
	}
	err = task.WriteResult(result)
	if err != nil {
		log.Panic(err)
	}
	e.TasksHandler.CountUpdatedTask()
	if e.TasksHandler.Len() == 1 {
		e.changeStatus(Completed)
	}
	return
}

type TaskStatus int

const (
	ReadyToCalc TaskStatus = iota
	Sent
	WaitingOtherTasks
	Calculated
)

type Task struct {
	PairID        int           `json:"id"`
	Arg1          interface{}   `json:"arg1"`
	Arg2          interface{}   `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operationTime"`
	result        int
	status        TaskStatus
	mut           sync.Mutex
}

func (t *Task) WriteResult(result int) error {
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.status == Sent {
		t.result = result
		t.status = Calculated
	} else if t.status == Calculated {
		return errors.New("BUG: разработчиком ожидается, что результат одной и той же задачи не может быть записан" +
			" больше одного раза")
	}
	return nil
}

func (t *Task) ChangeStatus(newStatus TaskStatus) {
	t.mut.Lock()
	defer t.mut.Unlock()
	if t.status != Calculated && t.status != newStatus {
		t.status = newStatus
	}
}

func (t *Task) IsReadyToCalc() bool {
	return t.status == ReadyToCalc
}
