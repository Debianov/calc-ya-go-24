package backend

import (
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
