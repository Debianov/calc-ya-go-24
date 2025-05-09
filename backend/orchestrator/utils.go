package main

import (
	"errors"
	"fmt"
	"github.com/Debianov/calc-ya-go-24/backend"
	"github.com/golang-jwt/jwt/v5"
	"log"
	"time"
)

const TodoSecretToDefendEnv = "not_under_deploy_not_under_deploy"

func GenerateJwt(user backend.CommonUser) (token string, err error) {
	var currentTime = time.Now()
	var jwtInstance = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": user.GetLogin(),
		"id":    user.GetId(),
		"nbf":   currentTime.Unix(),
		"exp":   currentTime.Add(10 * time.Minute).Unix(),
		"iat":   currentTime.Unix(),
	})
	token, err = jwtInstance.SignedString([]byte(TodoSecretToDefendEnv))
	if err != nil {
		log.Panic(err)
	}
	return
}

func ParseJwt(token string) (user backend.CommonUser, err error) {
	user = &backend.DbUser{}
	var tokenFromString *jwt.Token
	tokenFromString, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			panic(fmt.Errorf("метод подписи токена %v не ожидается", token.Header["alg"]))
		}
		return []byte(TodoSecretToDefendEnv), nil
	})
	if err != nil {
		return
	}
	claims, ok := tokenFromString.Claims.(jwt.MapClaims)
	if !ok {
		err = errors.New("не удалось прочитать структуру токена")
		return
	}
	user.SetLogin(claims["login"].(string))
	user.SetId(int64(claims["id"].(float64)))
	return
}
