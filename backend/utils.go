package backend

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"os"
)

type EnvVar struct {
	key                  string
	defaultValue         string
	extractedValue       string
	attemptToExtractFlag bool
}

func (e *EnvVar) Get() (result string, ok bool) {
	if e.extractedValue == "" && !e.attemptToExtractFlag {
		e.extractedValue = os.Getenv(e.key)
		e.attemptToExtractFlag = true
	}
	if e.extractedValue != "" {
		return e.extractedValue, true
	} else if e.defaultValue != "" {
		return e.defaultValue, true
	} else {
		return "", false
	}
}

func CallEnvVarFabric(key string, defaultValue string) *EnvVar {
	return &EnvVar{
		key:          key,
		defaultValue: defaultValue,
	}
}

func convertToInt64Interface(arg interface{}) (result interface{}, err error) {
	switch v := arg.(type) {
	case int:
		result = int64(v)
	case int32:
		result = int64(v)
	case int64, nil:
		result = v
	default:
		err = errors.New("arg должен быть числом")
	}
	return
}

type HashMan struct {
}

func (h *HashMan) Generate(salt string) (string, error) {
	var (
		saltedBytes = []byte(salt)
		hashedBytes []byte
		err         error
	)
	hashedBytes, err = bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

func (h *HashMan) Compare(hashedPassword string, possibleRelevantPassword string) (err error) {
	if err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(possibleRelevantPassword)); err != nil {
		err = errors.New("хеш не соответствует паролю, либо возникла другая ошибка")
	}
	return
}
