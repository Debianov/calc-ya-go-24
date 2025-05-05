package backend

import (
	"errors"
	jwt2 "github.com/golang-jwt/jwt/v5"
	"log"
	"os"
	"time"
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

const hmacSampleSecret = "test_test"

func GenerateJwt(user main.DbUser) (tokenBuf []byte, err error) {
	var currentTime = time.Now()
	var jwtInstance = jwt2.NewWithClaims(jwt2.SigningMethodHS256, jwt2.MapClaims{
		"login": user.GetLogin(),
		"id":    user.GetId(),
		"nbf":   currentTime.Unix(),
		"exp":   currentTime.Add(5 * time.Minute).Unix(),
		"iat":   currentTime.Unix(),
	})
	var token string
	token, err = jwtInstance.SignedString([]byte(hmacSampleSecret))
	if err != nil {
		log.Panic(err)
	}
	tokenBuf = []byte(token)
	return
}
