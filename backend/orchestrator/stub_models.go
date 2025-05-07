package main

import (
	"encoding/json"
	"errors"
	"github.com/Debianov/calc-ya-go-24/backend"
	"log"
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

type DbStub struct {
	lastId int64
	users  map[string]UserWithHashedPassword
}

func (s *DbStub) InsertUser(user UserWithHashedPassword) (lastId int64, err error) {
	s.users[user.GetLogin()] = user
	lastId++
	return lastId, nil
}

func (s *DbStub) SelectUser(login string) (user UserWithHashedPassword, err error) {
	v, ok := s.users[login]
	if !ok {
		err = errors.New("элемент не найден")
		return
	}
	return v, nil
}

func (s *DbStub) Flush() (err error) {
	s.users = make(map[string]UserWithHashedPassword)
	return
}

func (s *DbStub) Close() (err error) {
	return
}

func callStubDbFabric() *DbStub {
	return &DbStub{users: make(map[string]UserWithHashedPassword)}
}

func callStubDbWithRegisteredUserFabric(users ...UserStub) *DbStub {
	var (
		usersToStub = make(map[string]UserWithHashedPassword)
		err         error
	)
	for _, user := range users {
		err = user.SetHashedPassword(user.GetPassword())
		if err != nil {
			log.Panic(err)
		}
		usersToStub[user.GetLogin()] = &user
	}
	return &DbStub{users: usersToStub}
}

type UserStub struct {
	hashMan        backend.HashMan
	Login          string `json:"login"`
	Password       string `json:"password"`
	hashedPassword string
	id             int64
}

func (u *UserStub) Marshal() (result []byte, err error) {
	return json.Marshal(u)
}

func (u *UserStub) GetLogin() string {
	return u.Login
}

func (u *UserStub) SetLogin(s string) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetId() int64 {
	return u.id
}

func (u *UserStub) SetId(i int64) {
	//TODO implement me
	panic("implement me")
}

func (u *UserStub) GetPassword() string {
	return u.Password
}

func (u *UserStub) SetPassword(password string) {
	u.Password = password
}

func (u *UserStub) GetHashedPassword() string {
	return u.hashedPassword
}

func (u *UserStub) SetHashedPassword(salt string) (err error) {
	u.hashedPassword, err = u.hashMan.Generate(salt)
	return
}

func (u *UserStub) Is(user UserWithPassword) (status bool) {
	var (
		err error
	)
	if err = u.hashMan.Compare(u.GetHashedPassword(), user.GetPassword()); err != nil {
		return
	}
	status = true
	return
}

type parsedToken interface {
	backend.JsonPayload
	GetExpectedUser() CommonUser
}

type jwtTokenStub struct {
	ExpectedUser CommonUser
}

func (j *jwtTokenStub) Marshal() (result []byte, err error) {
	return json.Marshal(j)
}

func (j *jwtTokenStub) GetExpectedUser() CommonUser {
	return j.ExpectedUser
}
