package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Debianov/calc-ya-go-24/backend"
	"log"
)

type ExpressionsListStub struct {
	buf        []*backend.ExpressionStub
	exprOwners map[int64][]*backend.ExpressionStub
	cursor     int
}

func (s *ExpressionsListStub) AddExprFabric(fromUserId int64, postfix []string) (newExpr backend.CommonExpression, newExprId int) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionsListStub) GetAll() []backend.CommonExpression {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionsListStub) GetOwned(userOwnerId int64, exprId int) (backend.CommonExpression, bool) {
	userExprs := s.exprOwners[userOwnerId]
	for _, expr := range userExprs {
		if expr.GetId() == exprId {
			return expr, true
		}
	}
	return nil, false
}

func (s *ExpressionsListStub) GetAllOwned(userOwnerId int64) (result []backend.CommonExpression) {
	elems := s.exprOwners[userOwnerId]
	for _, v := range elems {
		result = append(result, v)
	}
	return
}

func (s *ExpressionsListStub) Remove(expr backend.CommonExpression) {
	//TODO implement me
	panic("implement me")
}

func (s *ExpressionsListStub) GetAllExprs() (result []backend.CommonExpression) {
	for _, expr := range s.buf {
		result = append(result, expr)
	}
	return
}

// Get получает ExpressionStub из buf по порядковому id Expression в этом buf.
func (s *ExpressionsListStub) Get(id int) (result backend.CommonExpression, ok bool) {
	if id < len(s.buf) {
		ok = true
		result = backend.CommonExpression(s.buf[id])
	} else {
		ok = false
	}
	return
}

func (s *ExpressionsListStub) GetReadyExpr() (result backend.CommonExpression) {
	var expr *backend.ExpressionStub
	for _, expr = range s.buf {
		if expr.GetStatus() == backend.Ready {
			result = expr
			return
		}
	}
	return nil
}

/*
callExprsListStubFabric формирует новый ExpressionsListStub, который может быть присвоен глобальной
переменной exprsList для подмены настоящего списка в целях тестирования.
Образует список с новым инициатором (ownerId), которому принадлежат expressions.
*/
func callExprsListStubFabric(ownerId int64, expressions ...backend.ExpressionStub) (result *ExpressionsListStub) {
	newExprsArray := make([]*backend.ExpressionStub, 0)
	for _, expr := range expressions {
		newExprsArray = append(newExprsArray, &expr)
	}
	result = &ExpressionsListStub{buf: newExprsArray,
		exprOwners: map[int64][]*backend.ExpressionStub{ownerId: newExprsArray}}
	return
}

func callExprsEmptyListFabric() (result *ExpressionsListStub) {
	return &ExpressionsListStub{buf: make([]*backend.ExpressionStub, 0), exprOwners: make(map[int64][]*backend.ExpressionStub)}
}

type DbStub struct {
	lastId int64
	users  map[string]backend.UserWithHashedPassword
	exprs  map[int64][]backend.ExpressionStub
}

func (s *DbStub) GetLastExprId() (int, error) {
	return 0, nil
}

func (s *DbStub) InsertExpr(expr backend.CommonExpression) (err error) {
	//TODO implement me
	panic("implement me")
}

func (s *DbStub) InsertUser(user backend.UserWithHashedPassword) (lastId int64, err error) {
	s.users[user.GetLogin()] = user
	lastId++
	return lastId, nil
}

func (s *DbStub) SelectUser(login string) (user backend.UserWithHashedPassword, err error) {
	v, ok := s.users[login]
	if !ok {
		err = errors.New("элемент не найден")
		return
	}
	return v, nil
}

func (s *DbStub) SelectAllExprs(userOwnerId int64) (exprs []backend.ShortExpression, err error) {
	fromExprs := s.exprs[userOwnerId]
	for _, v := range fromExprs {
		exprs = append(exprs, &v)
	}
	return
}

func (s *DbStub) SelectExpr(userOwnerId int64, exprId int) (expr backend.ShortExpression, err error) {
	userExprs := s.exprs[userOwnerId]
	for _, expr := range userExprs {
		if expr.GetId() == exprId {
			return &expr, nil
		}
	}
	return nil, fmt.Errorf("выражение ID %d у %d не найдено", exprId, userOwnerId)
}

func (s *DbStub) Flush() (err error) {
	s.users = make(map[string]backend.UserWithHashedPassword)
	s.exprs = make(map[int64][]backend.ExpressionStub)
	return
}

func (s *DbStub) InsertExprs(ownerId int64, exprs []backend.ExpressionStub) {
	s.exprs[ownerId] = exprs
}

func (s *DbStub) FlushExprs() (err error) {
	s.exprs = make(map[int64][]backend.ExpressionStub)
	return
}

func (s *DbStub) Close() (err error) {
	return
}

func callStubDbFabric() *DbStub {
	return &DbStub{users: make(map[string]backend.UserWithHashedPassword), exprs: make(map[int64][]backend.ExpressionStub)}
}

func callStubDbWithRegisteredUserFabric(users ...backend.UserStub) *DbStub {
	var (
		usersToStub = make(map[string]backend.UserWithHashedPassword)
		exprs       = make(map[int64][]backend.ExpressionStub)
		err         error
	)
	for _, user := range users {
		err = user.SetHashedPassword(user.GetPassword())
		if err != nil {
			log.Panic(err)
		}
		usersToStub[user.GetLogin()] = &user
	}
	return &DbStub{users: usersToStub, exprs: exprs}
}

type parsedToken interface {
	backend.JsonPayload
	GetExpectedUser() backend.CommonUser
}

type userForJwtToken struct {
	ExpectedUser backend.CommonUser
}

func (j *userForJwtToken) Marshal() (result []byte, err error) {
	return json.Marshal(j)
}

func (j *userForJwtToken) GetExpectedUser() backend.CommonUser {
	return j.ExpectedUser
}
