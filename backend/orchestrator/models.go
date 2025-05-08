package main

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	"iter"
	"maps"
	"slices"
	"sync"
)

type CommonUser interface {
	GetLogin() string
	SetLogin(string)
	GetId() int64
	SetId(int64)
}

type UserWithPassword interface {
	CommonUser
	GetPassword() string
	SetPassword(password string)
}

type UserWithHashedPassword interface {
	CommonUser
	GetHashedPassword() string
	SetHashedPassword(salt string) (err error)
	Is(password UserWithPassword) bool
}

/*
JsonUser -- структура для frontend-использования (контур frontend - backend).
*/
type JsonUser struct {
	id       int64
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (j *JsonUser) GetId() int64 {
	return j.id
}

func (j *JsonUser) SetId(newId int64) {
	j.id = newId
}

func (j *JsonUser) GetLogin() string {
	return j.Login
}

func (j *JsonUser) SetLogin(login string) {
	j.Login = login
}

func (j *JsonUser) GetPassword() string {
	return j.Password
}

func (j *JsonUser) SetPassword(password string) {
	j.Password = password
}

func CallJsonUserFabric() *JsonUser {
	return &JsonUser{}
}

/*
DbUser -- структура для внутреннего использования (контур db - backend).
*/
type DbUser struct {
	id             int64
	login          string
	hashedPassword string
	hashMan        backend.HashMan
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

// Is сравнивает пользовательские экземпляры по соответствию логина и пароля.
func (d *DbUser) Is(user UserWithPassword) (status bool) {
	var (
		err error
	)
	if user.GetLogin() != d.GetLogin() {
		return
	}
	if err = d.hashMan.Compare(d.GetHashedPassword(), user.GetPassword()); err != nil {
		return
	}
	status = true
	return
}

/*
wrapIntoDbUser устанавливает захешированный пароль, пригодный для хранения в db,
а также переносит login, используя данные jsonUser.
*/
func wrapIntoDbUser(jsonUser UserWithPassword) (instance *DbUser, err error) {
	instance = &DbUser{}
	instance.SetLogin(jsonUser.GetLogin())
	err = instance.SetHashedPassword(jsonUser.GetPassword())
	return
}

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
}

func (e *ExpressionsList) AddExprFabric(fromUserId int64, postfix []string) (newExpr backend.CommonExpression,
	newExprId int) {
	newExprId = e.generateId()
	newTaskSpace := backend.CallTasksHandlerFabric()
	newExpr = backend.CallExpressionFabric(postfix, newExprId, backend.Ready, newTaskSpace)
	newExpr.DivideIntoTasks()
	toAdd := newExpr.(*backend.Expression)
	e.mut.Lock()
	e.exprs[newExprId] = toAdd
	e.exprsOwners[fromUserId] = append(e.exprsOwners[fromUserId], toAdd)
	e.mut.Unlock()
	return
}

func (e *ExpressionsList) generateId() (id int) {
	e.mut.Lock()
	defer e.mut.Unlock()
	return len(e.exprs)
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

type RequestJson struct {
	JwtTokenJsonWrapper
	Expression string `json:"expression"`
}

func (r RequestJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}
