package main

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	"iter"
	"maps"
	"slices"
	"sync"
)

type JwtTokenJsonWrapper struct {
	Token string `json:"token"`
}

func (j *JwtTokenJsonWrapper) Marshal() (result []byte, err error) {
	return json.Marshal(j)
}

type CommonExpressionsList interface {
	AddExprFabric(fromUserId int64, postfix []string) (newExpr backend.CommonExpression, newExprId int)
	Get(exprId int) (backend.CommonExpression, bool)
	GetAll() []backend.CommonExpression
	GetOwned(userOwnerId int64, exprId int) (backend.CommonExpression, bool)
	GetAllOwned(userOwnerId int64) []backend.CommonExpression
	GetReadyExpr() (expr backend.CommonExpression)
	Remove(expr backend.CommonExpression)
}

type ExpressionsList struct {
	mut sync.Mutex
	/*
		exprs хранит только выполняющиеся выражения. Все посчитанные выражения отправляются в БД.
	*/
	exprs map[int]*backend.Expression
	/*
		exprsOwners отображает соответствия "пользователь - выражения". Хранит только
		выполняющиеся выражения. Все посчитанные выражения отправляются в БД.
	*/
	exprsOwners map[int64][]*backend.Expression
	/*
		idForNewExpr используется для нумерации выражений (generateId) согласно последнему выполненному выражению в БД.
	*/
	idForNewExpr int
}

func (e *ExpressionsList) AddExprFabric(fromUserId int64, postfix []string) (newExpr backend.CommonExpression,
	newExprId int) {
	newExprId = e.generateId()
	newTaskSpace := backend.CallTasksHandlerFabric()
	newExpr = backend.CallExpressionFabric(postfix, newExprId, fromUserId, backend.Ready, newTaskSpace)
	newExpr.DivideIntoTasks()
	toAdd := newExpr.(*backend.Expression)
	e.mut.Lock()
	e.exprs[newExprId] = toAdd
	e.exprsOwners[fromUserId] = append(e.exprsOwners[fromUserId], toAdd)
	e.idForNewExpr++
	e.mut.Unlock()
	return
}

func (e *ExpressionsList) generateId() (id int) {
	return e.idForNewExpr
}

/*
Get возвращает конкретное выражение по его id.
*/
func (e *ExpressionsList) Get(exprId int) (backend.CommonExpression, bool) {
	e.mut.Lock()
	defer e.mut.Unlock()
	var result, ok = e.exprs[exprId]
	return result, ok
}

/*
GetAll возвращает все выражения, хранящиеся в списке в рандомном порядке и без сортировки
по пользователям.
*/
func (e *ExpressionsList) GetAll() []backend.CommonExpression {
	e.mut.Lock()
	defer e.mut.Unlock()
	var (
		stop          func()
		v             *backend.Expression
		next          func() (*backend.Expression, bool)
		thereAreElems = true
		seq           iter.Seq[*backend.Expression]
		result        = make([]backend.CommonExpression, 0)
	)
	seq = maps.Values(e.exprs)
	next, stop = iter.Pull[*backend.Expression](seq)
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

/*
GetOwned возвращает конкретное значение и проверяет его принадлежность.
*/
func (e *ExpressionsList) GetOwned(userOwnerId int64, exprId int) (result backend.CommonExpression, ok bool) {
	e.mut.Lock()
	defer e.mut.Unlock()
	var exprFromList *backend.Expression
	exprFromList, ok = e.exprs[exprId]
	if ok && slices.Contains(e.exprsOwners[userOwnerId], exprFromList) {
		result = exprFromList
		return
	} else {
		ok = false
		result = nil
		return
	}
}

/*
GetAllOwned выдаёт значения в рандомном порядке все выражения, которые созданы пользователем с конкретным id.
*/
func (e *ExpressionsList) GetAllOwned(userOwnerId int64) (result []backend.CommonExpression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	for _, expr := range e.exprsOwners[userOwnerId] {
		result = append(result, expr)
	}
	return
}

func (e *ExpressionsList) GetReadyExpr() (expr backend.CommonExpression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	for _, v := range e.exprs {
		if v.GetStatus() == backend.Ready {
			return v
		}
	}
	return nil
}

func (e *ExpressionsList) Remove(expr backend.CommonExpression) {
	e.mut.Lock()
	defer e.mut.Unlock()
	delete(e.exprsOwners, expr.GetOwnerId())
	delete(e.exprs, expr.GetId())
}

func CallEmptyExpressionListFabric() *ExpressionsList {
	return &ExpressionsList{
		mut:         sync.Mutex{},
		exprs:       make(map[int]*backend.Expression),
		exprsOwners: make(map[int64][]*backend.Expression),
	}
}

func CallExpressionListWithLastIdFabric(lastId int) *ExpressionsList {
	return &ExpressionsList{
		mut:          sync.Mutex{},
		exprs:        make(map[int]*backend.Expression),
		exprsOwners:  make(map[int64][]*backend.Expression),
		idForNewExpr: lastId,
	}
}

type RequestJson struct {
	JwtTokenJsonWrapper
	Expression string `json:"expression"`
}

func (r RequestJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}
