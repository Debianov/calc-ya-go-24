package backend

import (
	"encoding/json"
	"go/types"
	"time"
)

type JsonPayload interface {
	Marshal() (result []byte, err error)
}

type RequestJson struct {
	Expression string `json:"expression"`
}

func (r RequestJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&r)
	return
}

/*
RequestNilJson изначально нужен для передачи nil и вызова Internal Server Error. Мы передаём nil, затем
он извлекается через Expression для создания Reader, а этот Reader запихивается в http.Request и передаётся
дальше в функцию. Далее, функция вызовет панику, паника перехватится PanicMiddleware, и далее по списку.

Используется в тесте TestBadGetHandler.
*/
type RequestNilJson struct {
	Expression types.Type `json:"expression"`
}

func (r RequestNilJson) Marshal() (result []byte, err error) {
	return nil, nil
}

type OKJson struct {
	Result float64 `json:"result"`
}

func (o OKJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&o)
	return
}

type ErrorJson struct {
	Error string `json:"error"`
}

func (e ErrorJson) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&e)
	return
}

type Expression struct {
	Expression []string
	ID         int    `json:"id""`
	Status     string `json:"status"`
	Result     int    `json:"result"`
}

func (t *Expression) DivideToParallelise() {
	// TODO
	return
}

func (t *Expression) Marshal() (result []byte, err error) {
	result, err = json.Marshal(&t)
	return
}

func (t *Expression) MarshalID() (result []byte, err error) {
	result, err = json.Marshal(&Expression{ID: t.ID})
	return
}

type Task struct {
	ID            int           `json:"id"`
	Arg1          int           `json:"arg1"`
	Arg2          int           `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operationTime"`
	Expression    *Expression
	result        int
}

type TaskIsDone struct {
	ID     int `json:"ID"`
	Result int `json:"result"`
}
