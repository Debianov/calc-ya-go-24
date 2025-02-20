package pkg

import (
	"strconv"
	"sync"
)

func IsNumber(token string) bool {
	if _, err := strconv.ParseFloat(token, 64); err == nil {
		return true
	}
	return false
}

func IsOperator(token string) bool {
	return token == "+" || token == "-" || token == "*" || token == "/"
}

type Stack[T any] struct {
	buf []T
	mut sync.Mutex
}

func (s *Stack[T]) Len() int {
	s.mut.Lock()
	defer s.mut.Unlock()
	return len(s.buf)
}

func (s *Stack[T]) Push(element T) {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.buf = append(s.buf, element)
}

func (s *Stack[T]) GetFirst() T {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.buf[0]
}

func (s *Stack[T]) GetLast() T {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.buf[len(s.buf)-1]
}

func (s *Stack[T]) Pop() T {
	result := s.GetLast()
	s.mut.Lock()
	s.buf = s.buf[:len(s.buf)-1]
	s.mut.Unlock()
	return result
}
