package backend

/*
Общие модели для агента и оркестратора
*/

import (
	"iter"
	"maps"
	"sync"
	"time"
)

type CommonTask interface {
	GetPairId() int32
	GetOperation() string
}

// GrpcTask должен реализовывать только TaskWithTime, его stub-ы и orchestrator.TaskToSend
type GrpcTask interface {
	CommonTask
	GetArg1() int64
	GetArg2() int64
	GetPermissibleDuration() string
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
	PopSentTask(taskId int) (InternalTask, time.Time, bool)
}

// TasksHandler - обёртка над pkg.Stack с дополнительными методами. Нужен для обработки случаев, когда несколько Task-ов готовы
// и нужно продолжить работу других Task-ов, зависящие от первых.
// В случае, когда все необходимые Task-и обновлены, их результаты записываются в зависимый Task, и дальше он отправляется
// для дальнейшей обработки.
// Для работы с TaskWithTime встроена отдельная структура.
type TasksHandler struct {
	sentTasks                          *sentTasksHandler
	buf                                []*Task
	tasksCountBeforeWaitingTask        int
	updatedTasksCountBeforeWaitingTask int
	mut                                sync.Mutex
}

func (t *TasksHandler) Add(task InternalTask) {
	t.mut.Lock()
	t.buf = append(t.buf, task.(*Task))
	t.mut.Unlock()
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

// RegisterFirst возвращает первую задачу, не удаляет её, но запоминает и не выдаёт повторно в дальнейшем.
// Удаляет в том случае, если задача не будет использоваться для вычисления других задач.
// Для простого получения задачи используйте Get.
func (t *TasksHandler) RegisterFirst() (task InternalTask) {
	task = t.Get(t.tasksCountBeforeWaitingTask)
	if task.IsReadyToCalc() {
		t.tasksCountBeforeWaitingTask++
		return
	} else {
		var expectedTask InternalTask
		if t.updatedTasksCountBeforeWaitingTask == t.tasksCountBeforeWaitingTask { // цикл в
			// горутине не требуется, поскольку агент будут самостоятельно тыкать в сервер, чтоб тот проверил на
			// наличие свободных таск
			switch t.tasksCountBeforeWaitingTask {
			case 1:
				if _, ok := task.GetArg1(); ok != true {
					expectedTask = t.Get(0)
					t.delete(0)
					task.SetArg1(expectedTask.GetResult())
				}
				t.updatedTasksCountBeforeWaitingTask = 0
				t.tasksCountBeforeWaitingTask = 0
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
				t.updatedTasksCountBeforeWaitingTask = 0
				t.tasksCountBeforeWaitingTask = 0
			default:
				if t.tasksCountBeforeWaitingTask < 3 {
					break
				}
				calculatedTaskOffset := t.tasksCountBeforeWaitingTask
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
				t.updatedTasksCountBeforeWaitingTask = t.updatedTasksCountBeforeWaitingTask - 2
				t.tasksCountBeforeWaitingTask = t.tasksCountBeforeWaitingTask - 2 + 1 // -2 удалённых и +1 текущий, который
				// теперь ReadyToCalc.
			}
			task.SetStatus(ReadyToCalc)
		}
		return
	}
}

// CountUpdatedTask обновляет число отправленных тасок. Обязателен к вызову, если любой Task, указатель которого
// хранится в экземпляре этой структуры, был обновлён.
func (t *TasksHandler) CountUpdatedTask() {
	t.updatedTasksCountBeforeWaitingTask++
}

func (t *TasksHandler) PopSentTask(taskId int) (InternalTask, time.Time, bool) {
	return t.sentTasks.PopSentTask(taskId)
}

// sentTasksHandler — map для работы с TaskWithTime структурой.
type sentTasksHandler struct {
	buf map[int]TaskWithTime
	mut sync.Mutex
}

func (t *sentTasksHandler) WrapWithTime(readyTask InternalTask, timeAtSendingTask time.Time) (result TaskWithTime) {
	result = TaskWithTime{
		task:              readyTask.(*Task),
		timeAtSendingTask: timeAtSendingTask,
	}
	t.mut.Lock()
	t.buf[int(readyTask.GetPairId())] = result
	t.mut.Unlock()
	return
}

func (t *sentTasksHandler) PopSentTask(taskId int) (*Task, time.Time, bool) {
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
		buf: make(map[int]TaskWithTime),
	}
}

func CallTasksHandlerFabric() *TasksHandler {
	newSentTasks := CallSentTasksFabric()
	return &TasksHandler{sentTasks: newSentTasks}
}

type CommonExpressionsList interface {
	AddExprFabric(postfix []string) (newExpr CommonExpression, newId int)
	GetAllExprs() []CommonExpression
	Get(id int) (CommonExpression, bool)
	GetReadyExpr() (expr CommonExpression)
}

type ExpressionsList struct {
	mut   sync.Mutex
	exprs map[int]*Expression
}

func (e *ExpressionsList) AddExprFabric(postfix []string) (newExpr CommonExpression, newId int) {
	newId = e.generateId()
	newTaskSpace := CallTasksHandlerFabric()
	newExpr = CallExpressionFabric(postfix, newId, Ready, newTaskSpace)
	newExpr.DivideIntoTasks()
	e.mut.Lock()
	e.exprs[newId] = newExpr.(*Expression)
	e.mut.Unlock()
	return
}

func (e *ExpressionsList) generateId() (id int) {
	e.mut.Lock()
	defer e.mut.Unlock()
	return len(e.exprs)
}

// GetAllExprs выдаёт значения в рандомном порядке.
func (e *ExpressionsList) GetAllExprs() []CommonExpression {
	e.mut.Lock()
	defer e.mut.Unlock()
	var (
		stop          func()
		v             *Expression
		next          func() (*Expression, bool)
		thereAreElems = true
		seq           iter.Seq[*Expression]
		result        = make([]CommonExpression, 0)
	)
	seq = maps.Values(e.exprs)
	next, stop = iter.Pull[*Expression](seq)
	defer stop()
	for {
		v, thereAreElems = next()
		if thereAreElems != false {
			result = append(result, v)
		} else {
			break
		}
	}
	return result
}

func (e *ExpressionsList) Get(id int) (CommonExpression, bool) {
	e.mut.Lock()
	var result, ok = e.exprs[id]
	e.mut.Unlock()
	return result, ok
}

func (e *ExpressionsList) GetReadyExpr() (expr CommonExpression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	for _, v := range e.exprs {
		if v.GetStatus() == Ready {
			return v
		}
	}
	return nil
}

func CallEmptyExpressionListFabric() *ExpressionsList {
	return &ExpressionsList{
		mut:   sync.Mutex{},
		exprs: make(map[int]*Expression),
	}
}
