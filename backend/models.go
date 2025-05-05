package backend

/*
Общие модели для агента и оркестратора
*/

import (
	"context"
	"database/sql"
	"errors"
	"iter"
	"maps"
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

// TasksHandler - обёртка над pkg.Stack с дополнительными методами. Нужен для обработки случаев, когда несколько Task-ов готовы
// и нужно продолжить работу других Task-ов, зависящие от первых.
// В случае, когда все необходимые Task-и обновлены, их результаты записываются в зависимый Task, и дальше он отправляется
// для дальнейшей обработки.
// Для работы с TaskWithTime встроена отдельная структура.
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

type CommonUser interface {
	GetLogin() string
	SetLogin(string)
	GetId() int64
	SetId(int64)
}

type UserWithHashedPassword interface {
	CommonUser
	GetHashedPassword() string
	SetHashedPassword(salt string) (err error)
}

type DbUser struct {
	id             int64
	login          string
	hashedPassword string
	hashMan        HashMan
}

func (d *DbUser) GetId() int64 {
	return d.id
}

func (d *DbUser) SetId(newId int64) {
	d.id = newId
}

func (d *DbUser) GetLogin() string {
	return d.login
}

func (d *DbUser) SetLogin(login string) {
	d.login = login
}

func (d *DbUser) GetHashedPassword() string {
	return d.hashedPassword
}

/*
SetHashedPassword генерирует по salt и устанавливает захешированный пароль.
*/
func (d *DbUser) SetHashedPassword(salt string) (err error) {
	d.hashedPassword, err = d.hashMan.Generate(salt)
	return
}

/*
CallDbUserFabric устанавливает захешированный пароль, пригодный для хранения в db,
а также переносит login, используя данные jsonUser.
*/
func CallDbUserFabric(jsonUser UserWithPassword) (instance *DbUser, err error) {
	instance = &DbUser{}
	instance.SetLogin(jsonUser.GetLogin())
	err = instance.SetHashedPassword(jsonUser.GetPassword())
	return
}

type Db struct {
	ctx     context.Context
	innerDb *sql.DB
}

func (d *Db) InsertUser(user UserWithHashedPassword) (lastId int64, err error) {
	var (
		query = `
	INSERT INTO users (name, password) values ($1, $2)
	`
		result sql.Result
	)
	result, err = d.innerDb.ExecContext(d.ctx, query, user.GetLogin(), user.GetHashedPassword())
	if err != nil {
		return
	}
	lastId, err = result.LastInsertId()
	return
}

func CallDbFabric() *Db {
	var (
		innerDb = GetDefaultSqlServer()
		ctx     = context.TODO()
	)
	innerDb.PingContext(ctx)
	return &Db{ctx: context.TODO(), innerDb: innerDb}
}
