package backend

import (
	"encoding/json"
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

type Expression struct {
	Postfix []string
	ID      int        `json:"id"`
	Status  ExprStatus `json:"status"`
	Result  int        `json:"result"`
	tasks   pkg.Stack[Task]
	mut     sync.Mutex
}

func (e *Expression) DivideIntoTasks() {
	var (
		operand               string
		operandsBeforeOperand []int64
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
				newId   = e.generateId()
				newTask *Task
			)
			if len(operandsBeforeOperand) == 2 {
				newTask = &Task{ID: newId, Arg1: operandsBeforeOperand[0], Arg2: operandsBeforeOperand[1],
					Operation: r, OperationTime: e.getOperationTime(r), isReady: true}
			} else if len(operandsBeforeOperand) == 1 {
				newTask = &Task{ID: newId, Arg2: operandsBeforeOperand[0], Operation: r,
					OperationTime: e.getOperationTime(r)}
			} else {
				newTask = &Task{ID: newId, Operation: r, OperationTime: e.getOperationTime(r)}
			}
			e.tasks.Push(*newTask)
			operandsBeforeOperand = make([]int64, 0)
		}
	}
	return
}

func (e *Expression) generateId() int {
	return e.tasks.Len() - 1 // TODO нужно в один id впихнуть два
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

func (e *Expression) GetReadyToSendTask() (result *Task) {
	task := e.tasks.GetFirst()
	if task.isReadyToSend() {
		*result = e.tasks.Pop()
		defer e.checkLastTaskAndChangeStatus()
		return result
	} else {
		e.changeStatus(NoReadyTasks)
		return nil
	}
}

func (e *Expression) checkLastTaskAndChangeStatus() {
	toCheck := e.tasks.GetFirst()
	if !toCheck.isReadyToSend() {
		e.changeStatus(NoReadyTasks)
	} else {
		e.changeStatus(Ready)
	}
}

func (e *Expression) changeStatus(status ExprStatus) {
	e.mut.Lock()
	defer e.mut.Unlock()
	e.Status = status
}

func (e *Expression) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

func (e *Expression) MarshalID() (result []byte, err error) {
	result, err = json.Marshal(&Expression{ID: e.ID})
	return
}

type Task struct {
	ID            int           `json:"id"`
	Arg1          int64         `json:"arg1"`
	Arg2          int64         `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operationTime"`
	isReady       bool
	result        int
}

func (t *Task) isReadyToSend() bool {
	return t.isReady
}
