package backend

/*
Общие модели для агента и оркестратора
*/

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/pkg"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type PairIdHolder interface {
	GetPairId() int32
}

type ResultHolder interface {
	GetResult() int64
}

type CommonTask interface {
	PairIdHolder
	GetOperation() string
}

// GrpcTask должен реализовывать только TaskWithTime, его stub-ы и orchestrator.TaskToSend
type GrpcTask interface {
	CommonTask
	GetArg1() int64
	GetArg2() int64
	GetPermissibleDuration() string
}

type GrpcResult interface {
	PairIdHolder
	ResultHolder
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
	return t.task.arg1.(int64)
}

func (t *TaskWithTime) GetArg2() int64 {
	return t.task.arg2.(int64)
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

type CommonTasksHandler interface {
	Add(task InternalTask)
	Get(ind int) InternalTask
	Len() int
	RegisterFirst() (task InternalTask)
	CountUpdatedTask()
	PopSentTask(taskId int32) (InternalTask, time.Time, bool)
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
TasksHandler - обёртка над pkg.Stack с дополнительными методами. Нужен для обработки случаев, когда несколько
Task-ов готовы и нужно продолжить работу других Task-ов, зависящие от первых. В случае, когда все необходимые Task-и
обновлены, их результаты записываются в зависимый Task, и дальше он отправляется для дальнейшей обработки.
Для работы с TaskWithTime встроена отдельная структура.
*/
type TasksHandler struct {
	sentTasks                          *sentTasksHandler
	buf                                []*Task
	tasksCountBeforeWaitingTask        atomic.Value
	updatedTasksCountBeforeWaitingTask atomic.Value
	mut                                sync.Mutex
}

func (t *TasksHandler) Add(task InternalTask) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.buf = append(t.buf, task.(*Task))
}

func (t *TasksHandler) Get(ind int) InternalTask {
	t.mut.Lock()
	defer t.mut.Unlock()
	return t.buf[ind]
}

func (t *TasksHandler) delete(ind int) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.buf = append(t.buf[:ind], t.buf[ind+1:]...)
}

func (t *TasksHandler) Len() int {
	t.mut.Lock()
	defer t.mut.Unlock()
	return len(t.buf)
}

func (t *TasksHandler) getTasksCountBeforeWaitingTask() int {
	return t.tasksCountBeforeWaitingTask.Load().(int)
}

func (t *TasksHandler) getUpdatedTasksCountBeforeWaitingTask() int {
	return t.updatedTasksCountBeforeWaitingTask.Load().(int)
}

// RegisterFirst возвращает первую задачу, не удаляет её, но запоминает и не выдаёт повторно в дальнейшем.
// Удаляет в том случае, если задача не будет использоваться для вычисления других задач.
// Для простого получения задачи используйте Get.
func (t *TasksHandler) RegisterFirst() (task InternalTask) {
	task = t.Get(t.getTasksCountBeforeWaitingTask())
	if task.IsReadyToCalc() {
		t.addTasksCountBeforeWaitingTask(1)
		return
	} else {
		var expectedTask InternalTask
		if t.getUpdatedTasksCountBeforeWaitingTask() == t.getTasksCountBeforeWaitingTask() { // цикл в
			// горутине не требуется, поскольку агент будут самостоятельно тыкать в сервер, чтоб тот проверил на
			// наличие свободных таск
			switch t.getTasksCountBeforeWaitingTask() {
			case 1:
				if _, ok := task.GetArg1(); ok != true {
					expectedTask = t.Get(0)
					t.delete(0)
					task.SetArg1(expectedTask.GetResult())
				}
				t.updatedTasksCountBeforeWaitingTask.Store(0)
				t.tasksCountBeforeWaitingTask.Store(0)
			case 2:
				if _, ok := task.GetArg1(); ok != true {
					expectedTask = t.Get(0)
					t.delete(0)
					task.SetArg1(expectedTask.GetResult())
				}
				if _, ok := task.GetArg2(); ok != true {
					expectedTask = t.Get(0)
					t.delete(0)
					task.SetArg2(expectedTask.GetResult())
				}
				t.updatedTasksCountBeforeWaitingTask.Store(0)
				t.tasksCountBeforeWaitingTask.Store(0)
			default:
				if t.getTasksCountBeforeWaitingTask() < 3 {
					break
				}
				calculatedTaskOffset := t.getTasksCountBeforeWaitingTask()
				if _, ok := task.GetArg2(); ok != true {
					expectedTask = t.Get(calculatedTaskOffset - 1)
					t.delete(calculatedTaskOffset - 1)
					task.SetArg2(expectedTask.GetResult())
				}
				if _, ok := task.GetArg1(); ok != true {
					expectedTask = t.Get(calculatedTaskOffset - 2)
					t.delete(calculatedTaskOffset - 2)
					task.SetArg1(expectedTask.GetResult())
				}
				t.updatedTasksCountBeforeWaitingTask.Store(t.getUpdatedTasksCountBeforeWaitingTask() - 2)
				t.tasksCountBeforeWaitingTask.Store(t.getTasksCountBeforeWaitingTask() - 2 + 1) // -2 удалённых и
				// +1 текущий, который теперь ReadyToCalc.
			}
			task.SetStatus(ReadyToCalc)
		}
		return
	}
}

func (t *TasksHandler) addTasksCountBeforeWaitingTask(delta int) {
	t.tasksCountBeforeWaitingTask.Store(t.getTasksCountBeforeWaitingTask() + delta)
}

func (t *TasksHandler) addUpdatedTasksCountBeforeWaitingTasks(delta int) {
	t.updatedTasksCountBeforeWaitingTask.Store(t.getUpdatedTasksCountBeforeWaitingTask() + delta)
}

// CountUpdatedTask обновляет число отправленных тасок. Обязателен к вызову, если любой Task, указатель которого
// хранится в экземпляре этой структуры, был обновлён.
func (t *TasksHandler) CountUpdatedTask() {
	t.addUpdatedTasksCountBeforeWaitingTasks(1)
}

func (t *TasksHandler) PopSentTask(taskId int32) (InternalTask, time.Time, bool) {
	return t.sentTasks.PopSentTask(taskId)
}

// sentTasksHandler — map для работы с TaskWithTime структурой.
type sentTasksHandler struct {
	buf map[int32]TaskWithTime
	mut sync.Mutex
}

func (t *sentTasksHandler) WrapWithTime(readyTask InternalTask, timeAtSendingTask time.Time) (result TaskWithTime) {
	result = TaskWithTime{
		task:              readyTask.(*Task),
		timeAtSendingTask: timeAtSendingTask,
	}
	t.mut.Lock()
	t.buf[readyTask.GetPairId()] = result
	t.mut.Unlock()
	return
}

func (t *sentTasksHandler) PopSentTask(taskId int32) (*Task, time.Time, bool) {
	t.mut.Lock()
	taskWithTime, ok := t.buf[taskId]
	if ok {
		delete(t.buf, taskId)
	}
	t.mut.Unlock()
	return taskWithTime.GetWrappedTask().(*Task), taskWithTime.GetTimeAtSendingTask(), ok
}

func CallSentTasksFabric() *sentTasksHandler {
	return &sentTasksHandler{
		buf: make(map[int32]TaskWithTime),
	}
}

func CallTasksHandlerFabric() (newInstance *TasksHandler) {
	newSentTasks := CallSentTasksFabric()
	newInstance = &TasksHandler{sentTasks: newSentTasks}
	newInstance.updatedTasksCountBeforeWaitingTask.Store(0)
	newInstance.tasksCountBeforeWaitingTask.Store(0)
	return
}

// JsonPayload реализует тот же интерфейс, что и json.Marshaler, только метод с другим названием
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
	task, timeAtSendingTask, ok := e.tasksHandler.PopSentTask(result.GetPairId())
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
