package main

import (
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
