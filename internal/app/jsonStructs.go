package app

import (
	"encoding/json"
	"go/types"
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
