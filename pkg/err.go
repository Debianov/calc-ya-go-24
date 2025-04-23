package pkg

import "errors"

var (
	mismatchedParentheses = errors.New("mismatched parentheses")
	InvalidExpression     = errors.New("invalid expression")
)
