package backend

import (
	"iter"
	"maps"
	"slices"
	"sync"
	"time"
)

// Tasks - обёртка над pkg.Stack с дополнительными методами. Нужен для обработки случаев, когда несколько Task-ов готовы
// и нужно продолжить работу других Task-ов, зависящие от первых.
// В случае, когда все необходимые Task-и обновлены, их результаты записываются в нужный Task, и дальше он отправляется
// для дальнейшей обработки.
// Для работы с TaskToSend встроена структура.
type Tasks struct {
	*sentTasks
	buf                                []*Task
	tasksCountBeforeWaitingTask        int
	updatedTasksCountBeforeWaitingTask int
	mut                                sync.Mutex
}

func (t *Tasks) add(task *Task) {
	t.mut.Lock()
	t.buf = append(t.buf, task)
	t.mut.Unlock()
}

func (t *Tasks) Get(ind int) *Task {
	t.mut.Lock()
	defer t.mut.Unlock()
	return t.buf[ind]
}

func (t *Tasks) delete(ind int) {
	t.mut.Lock()
	defer t.mut.Unlock()
	slices.Delete(t.buf, ind, ind+1)
}

func (t *Tasks) Len() int {
	t.mut.Lock()
	defer t.mut.Unlock()
	return len(t.buf)
}

// registerFirst возвращает первую задачу, не удаляет её, но регистрирует и не выдаёт ту же задачу в дальнейшем.
// Удаляет в том случае, если задача не используется для вычисления других задач.
// Для простого получения задачи используйте Get.
func (t *Tasks) registerFirst() (task *Task) {
	task = t.Get(t.tasksCountBeforeWaitingTask)
	if task.IsReadyToCalc() {
		t.tasksCountBeforeWaitingTask++
		return
	} else {
		var expectedTask Task
		if t.updatedTasksCountBeforeWaitingTask == t.tasksCountBeforeWaitingTask { // цикл в
			// горутине не требуется, поскольку агент будут самостоятельно тыкать в сервер, чтоб тот проверил на
			// наличие свободных таск
			if _, ok := task.Arg2.(string); ok == true {
				expectedTask = *t.Get(0)
				t.delete(0)
				task.Arg2 = int64(expectedTask.result)
			}
			if _, ok := task.Arg1.(string); ok == true {
				expectedTask = *t.Get(0)
				t.delete(0)
				task.Arg1 = int64(expectedTask.result)
			}
			task.ChangeStatus(ReadyToCalc)
			t.updatedTasksCountBeforeWaitingTask = 0
			t.tasksCountBeforeWaitingTask = 0
		}
		return
	}
}

// CountUpdatedTask обновляет число отправленных тасок. Обязателен к вызову, если любой Task, указатель которого
// хранится в экземпляре этой структуры, был обновлён.
func (t *Tasks) CountUpdatedTask() {
	t.updatedTasksCountBeforeWaitingTask++
}

// sentTasks — map для работы с TaskToSend структурой.
type sentTasks struct {
	buf map[int]TaskToSend
	mut sync.Mutex
}

func (t *sentTasks) TaskToSendFabricAdd(readyTask *Task, timeAtSendingTask time.Time) (result TaskToSend) {
	result = TaskToSend{
		Task:              readyTask,
		timeAtSendingTask: timeAtSendingTask,
	}
	t.mut.Lock()
	t.buf[readyTask.PairID] = result
	t.mut.Unlock()
	return
}

func (t *sentTasks) getSentTask(taskId int) (*Task, time.Time, bool) {
	t.mut.Lock()
	taskWithTimer, ok := t.buf[taskId]
	if ok {
		delete(t.buf, taskId)
	}
	t.mut.Unlock()
	return taskWithTimer.Task, taskWithTimer.timeAtSendingTask, ok
}

func sentTasksFabric() *sentTasks {
	return &sentTasks{
		buf: make(map[int]TaskToSend),
	}
}

func TasksFabric() *Tasks {
	newSentTasks := sentTasksFabric()
	return &Tasks{sentTasks: newSentTasks}
}

type ExpressionsList struct {
	mut   sync.Mutex
	exprs map[int]*Expression
}

func (e *ExpressionsList) ExprFabricAdd(postfix []string) (newExpr *Expression, newId int) {
	newId = e.generateId()
	newTaskSpace := TasksFabric()
	newExpr = &Expression{Postfix: postfix, ID: newId, Status: Ready, TasksHandler: newTaskSpace}
	newExpr.DivideIntoTasks()
	e.mut.Lock()
	e.exprs[newId] = newExpr
	e.mut.Unlock()
	return
}

func (e *ExpressionsList) generateId() (id int) {
	e.mut.Lock()
	defer e.mut.Unlock()
	return len(e.exprs)
}

// GetAllExprs выдаёт значения в рандомном порядке.
func (e *ExpressionsList) GetAllExprs() []*Expression {
	e.mut.Lock()
	defer e.mut.Unlock()
	var (
		stop          func()
		v             *Expression
		next          func() (*Expression, bool)
		thereAreElems = true
		seq           iter.Seq[*Expression]
		result        = make([]*Expression, 0)
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

func (e *ExpressionsList) Get(id int) (*Expression, bool) {
	e.mut.Lock()
	var result, ok = e.exprs[id]
	e.mut.Unlock()
	return result, ok
}

func (e *ExpressionsList) GetReadyExpr() (expr *Expression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	for _, v := range e.exprs {
		if v.Status == Ready {
			return v
		}
	}
	return nil
}

func ExpressionListEmptyFabric() *ExpressionsList {
	return &ExpressionsList{
		mut:   sync.Mutex{},
		exprs: make(map[int]*Expression),
	}
}

func ExpressionListFabricWithElements(exprs []*Expression) *ExpressionsList {
	var result = make(map[int]*Expression)
	for _, expr := range exprs {
		result[expr.ID] = expr
	}
	return &ExpressionsList{
		mut:   sync.Mutex{},
		exprs: result,
	}
}
