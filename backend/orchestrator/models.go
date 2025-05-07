package main

import (
	"encoding/json"
	"github.com/Debianov/calc-ya-go-24/backend"
	"iter"
	"maps"
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
	AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int)
	GetAllExprs() []backend.CommonExpression
	Get(id int) (backend.CommonExpression, bool)
	GetReadyExpr() (expr backend.CommonExpression)
}

type ExpressionsList struct {
	mut   sync.Mutex
	exprs map[int]*backend.Expression
}

func (e *ExpressionsList) AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int) {
	newId = e.generateId()
	newTaskSpace := backend.CallTasksHandlerFabric()
	newExpr = backend.CallExpressionFabric(postfix, newId, backend.Ready, newTaskSpace)
	newExpr.DivideIntoTasks()
	e.mut.Lock()
	e.exprs[newId] = newExpr.(*backend.Expression)
	e.mut.Unlock()
	return
}

func (e *ExpressionsList) generateId() (id int) {
	e.mut.Lock()
	defer e.mut.Unlock()
	return len(e.exprs)
}

// GetAllExprs выдаёт значения в рандомном порядке.
func (e *ExpressionsList) GetAllExprs() []backend.CommonExpression {
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

func (e *ExpressionsList) Get(id int) (backend.CommonExpression, bool) {
	e.mut.Lock()
	var result, ok = e.exprs[id]
	e.mut.Unlock()
	return result, ok
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

func CallEmptyExpressionListFabric() *ExpressionsList {
	return &ExpressionsList{
		mut:   sync.Mutex{},
		exprs: make(map[int]*backend.Expression),
	}
}
