package main

import (
	"github.com/golang-jwt/jwt/v5"
	"log"
	"time"
)

const hmacSampleSecret = "test_test"

func GenerateJwt(user DbUser) (tokenBuf []byte, err error) {
	var currentTime = time.Now()
	var jwtInstance = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
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
