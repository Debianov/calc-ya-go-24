package main

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
)

type ExpressionsListStub struct {
	buf    []backend.ExpressionStub
	cursor int
}

func (s *ExpressionsListStub) AddExprFabric(postfix []string) (newExpr backend.CommonExpression, newId int) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionsListStub) GetAllExprs() (result []backend.CommonExpression) {
	for _, expr := range s.buf {
		result = append(result, backend.CommonExpression(&expr))
	}
	return
}

// Get получает ExpressionStub из buf по порядковому id Expression в этом buf.
func (s *ExpressionsListStub) Get(id int) (result backend.CommonExpression, ok bool) {
	if id < len(s.buf) {
		ok = true
		result = backend.CommonExpression(&s.buf[id])
	} else {
		ok = false
	}
	return
}

func (s *ExpressionsListStub) GetReadyExpr() (result backend.CommonExpression) {
	var expr backend.ExpressionStub
	for _, expr = range s.buf {
		if expr.GetStatus() == backend.Ready {
			result = &expr
			return
		}
	}
	return nil
}

/*
callExprsListStubFabric формирует новый ExpressionsListStub, который может быть присвоен глобальной
переменной exprsList для подмены настоящего списка в целях тестирования, или использоваться как-то ещё.
*/
func callExprsListStubFabric(expressions ...backend.ExpressionStub) (result *ExpressionsListStub) {
	if len(expressions) == 0 {
		result = &ExpressionsListStub{}
	} else {
		result = &ExpressionsListStub{buf: expressions}
	}
	return
}

type StubDb struct {
	lastId int64
	users  map[string]UserWithHashedPassword
}

func (s *StubDb) InsertUser(user UserWithHashedPassword) (lastId int64, err error) {
	s.users[user.GetLogin()] = user
	lastId++
	return lastId, nil
}

func (s *StubDb) SelectUser(login string) (user *DbUser, err error) {
	v, ok := s.users[login]
	if !ok {
		err = errors.New("элемент не найден")
		return
	}
	return v.(*DbUser), nil
}

func (s *StubDb) Flush() (err error) {
	s.users = make(map[string]UserWithHashedPassword)
	return
}

func (s *StubDb) Close() (err error) {
	return
}

func callStubDbFabric() *StubDb {
	return &StubDb{users: make(map[string]UserWithHashedPassword)}
}

func callStubDbWithRegisteredUserFabric(users ...UserWithHashedPassword) *StubDb {
	var usersToStub = make(map[string]UserWithHashedPassword)
	for _, user := range users {
		usersToStub[user.GetLogin()] = user
	}
	return &StubDb{users: usersToStub}
}

type UserStub struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	id       int64
}

func (u *UserStub) Marshal() (result []byte, err error) {
	return json.Marshal(u)
}

func (u *UserStub) GetLogin() string {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) SetLogin(s string) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetId() int64 {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) SetId(i int64) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetHashedPassword() string {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) SetHashedPassword(salt string) (err error) {
	//TODO implement me
	panic("implement me")
}

type jwtTokenStub struct {
}

func (j *jwtTokenStub) Marshal() (result []byte, err error) {
	return json.Marshal(j)
}
