package pkg

import (
	"errors"
	"strconv"
	"strings"
)

type Stack[T any] struct {
	buf []T
}

func (s *Stack[T]) len() int {
	return len(s.buf)
}

func (s *Stack[T]) push(element T) {
	s.buf = append(s.buf, element)
}

func (s *Stack[T]) getLast() T {
	return s.buf[len(s.buf)-1]
}

func (s *Stack[T]) pop() T {
	result := s.getLast()
	s.buf = s.buf[:len(s.buf)-1]
	return result
}

func Calc(expression string) (float64, error) {
	if len(expression) == 0 {
		return 0, nil
	}
	tokens := tokenize(expression)
	postfix, err := translateToPostfix(tokens)
	if err != nil {
		return 0, err
	}
	return evaluatePostfix(postfix)
}

func tokenize(expr string) []string {
	var (
		tokens       []string
		currentToken strings.Builder
	)

	for _, char := range expr {
		switch char {
		case ' ':
			continue
		case '+', '-', '*', '/', '(', ')':
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			tokens = append(tokens, string(char))
		default:
			currentToken.WriteRune(char)
		}
	}

	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

func translateToPostfix(tokens []string) ([]string, error) {
	var (
		output              []string
		operators           = Stack[string]{make([]string, 0)}
		operandCount        int
		operatorCount       int
		firstMustBeOperator bool // после любой ) должен идти только оператор. С помощью этого флага мы будем проверять
		// на наличие этого условия.
	)

	for _, token := range tokens {
		if isNumber(token) {
			if firstMustBeOperator {
				return nil, invalidExpression
			}
			output = append(output, token)
			operandCount++
		} else if token == "(" {
			if firstMustBeOperator {
				return nil, invalidExpression
			}
			operators.push(token)
		} else if token == ")" {
			for operators.len() > 0 && operators.getLast() != "(" {
				output = append(output, operators.pop())
			}
			if operators.len() == 0 {
				return nil, mismatchedParentheses
			}
			firstMustBeOperator = true
			operators.pop()
		} else if isOperator(token) {
			if firstMustBeOperator {
				firstMustBeOperator = false
			}
			for operators.len() > 0 && getPriority(operators.getLast()) >= getPriority(token) {
				output = append(output, operators.pop())
			}
			operators.push(token)
			operatorCount++
		} else {
			return nil, errors.New("invalid operator/operand")
		}
	}

	for operators.len() > 0 {
		if operators.getLast() == "(" {
			return nil, mismatchedParentheses
		}
		output = append(output, operators.pop())
	}

	if operatorCount != operandCount-1 {
		return nil, invalidExpression
	}

	return output, nil
}

func evaluatePostfix(postfix []string) (float64, error) {
	var stack = Stack[float64]{make([]float64, 0)}

	for _, token := range postfix {
		if isNumber(token) {
			num, _ := strconv.ParseFloat(token, 64)
			stack.push(num)
		} else if isOperator(token) {
			b := stack.pop()
			a := stack.pop()

			switch token {
			case "+":
				stack.push(a + b)
			case "-":
				stack.push(a - b)
			case "*":
				stack.push(a * b)
			case "/":
				if b == 0 {
					return 0, errors.New("division by zero")
				}
				stack.push(a / b)
			}
		}
	}

	return stack.getLast(), nil
}

func isNumber(token string) bool {
	if _, err := strconv.ParseFloat(token, 64); err == nil {
		return true
	}
	return false
}

func isOperator(token string) bool {
	return token == "+" || token == "-" || token == "*" || token == "/"
}

func getPriority(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	default:
		return 0
	}
}
