package backend

import (
	"errors"
)

type CasesHandler interface {
	GetHttpMethod() string
	GetUrlTarget() string
	GetExpectedHttpCode() int
}

type HttpCasesHandler[K, V JsonPayload] struct {
	RequestsToSend    []K
	ExpectedResponses []V
	HttpMethod        string
	UrlTarget         string
	ExpectedHttpCode  int
}

func (h *HttpCasesHandler[K, V]) GetHttpMethod() string {
	return h.HttpMethod
}

func (h *HttpCasesHandler[K, V]) GetUrlTarget() string {
	return h.UrlTarget
}

func (h *HttpCasesHandler[K, V]) GetExpectedHttpCode() int {
	return h.ExpectedHttpCode
}

type ServerMuxHttpCasesHandler[K, V JsonPayload] struct {
	RequestsToSend    []K
	ExpectedResponses []V
	HttpMethod        string
	UrlTemplate       string
	UrlTarget         string
	ExpectedHttpCode  int
}

func (s *ServerMuxHttpCasesHandler[K, V]) GetHttpMethod() string {
	return s.HttpMethod
}

func (s *ServerMuxHttpCasesHandler[K, V]) GetUrlTarget() string {
	return s.UrlTarget
}

func (s *ServerMuxHttpCasesHandler[K, V]) GetExpectedHttpCode() int {
	return s.ExpectedHttpCode
}

type ByteCase struct {
	ToSend   []byte
	Expected []byte
}

func ConvertToByteCases[K, V JsonPayload](reqs []K, resps []V) (result []ByteCase, err error) {
	if len(reqs) != len(resps) {
		err = errors.New("reqs и resps должны быть одной длины")
		return
	}
	for ind := 0; ind < len(reqs); ind++ {
		var (
			reqBuf  []byte
			respBuf []byte
		)
		reqBuf, err = reqs[ind].Marshal()
		if err != nil {
			return
		}
		respBuf, err = resps[ind].Marshal()
		if err != nil {
			return
		}
		result = append(result, ByteCase{ToSend: reqBuf, Expected: respBuf})
	}
	return
}
