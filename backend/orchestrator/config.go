package main

import (
	"database/sql"
	"log"
	"net/http"
)

func GetDefaultHttpServer(handler http.Handler) *http.Server {
	return &http.Server{Addr: "127.0.0.1:8000", Handler: handler}
}

func GetDefaultGrpcServer() *GrpcTaskServer {
	return &GrpcTaskServer{Addr: "127.0.0.1:5000"}
}

func GetDefaultSqlServer() *sql.DB {
	var db, err = sql.Open("sqlite3", "calc.db")
	if err != nil {
		log.Panic(err)
	}
	return db
}

func GetTestSqlServer() *sql.DB {
	var db, err = sql.Open("sqlite3", "testCalc.db")
	if err != nil {
		log.Panic(err)
	}
	return db
}
