package backend

import (
	"errors"
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
