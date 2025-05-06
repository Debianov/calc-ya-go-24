package main

import "github.com/Debianov/calc-ya-go-24/backend"

type CommonUser interface {
	GetLogin() string
	SetLogin(string)
	GetId() int64
	SetId(int64)
}

type UserWithHashedPassword interface {
	CommonUser
	GetHashedPassword() string
	SetHashedPassword(salt string) (err error)
}

type UserWithPassword interface {
	CommonUser
	GetPassword() string
	SetPassword(password string)
}

/*
JsonUser -- структура для frontend-использования (контур frontend - backend).
*/
type JsonUser struct {
	id       int64
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (j *JsonUser) GetId() int64 {
	return j.id
}

func (j *JsonUser) SetId(newId int64) {
	j.id = newId
}

func (j *JsonUser) GetLogin() string {
	return j.Login
}

func (j *JsonUser) SetLogin(login string) {
	j.Login = login
}

func (j *JsonUser) GetPassword() string {
	return j.Password
}

func (j *JsonUser) SetPassword(password string) {
	j.Password = password
}

func CallJsonUserFabric() *JsonUser {
	return &JsonUser{}
}

/*
DbUser -- структура для внутреннего использования (контур db - backend).
*/
type DbUser struct {
	id             int64
	login          string
	hashedPassword string
	hashMan        backend.HashMan
}

func (d *DbUser) GetId() int64 {
	return d.id
}

func (d *DbUser) SetId(newId int64) {
	d.id = newId
}

func (d *DbUser) GetLogin() string {
	return d.login
}

func (d *DbUser) SetLogin(login string) {
	d.login = login
}

func (d *DbUser) GetHashedPassword() string {
	return d.hashedPassword
}

/*
SetHashedPassword генерирует по salt и устанавливает захешированный пароль.
*/
func (d *DbUser) SetHashedPassword(salt string) (err error) {
	d.hashedPassword, err = d.hashMan.Generate(salt)
	return
}

/*
CallDbUserFabric устанавливает захешированный пароль, пригодный для хранения в db,
а также переносит login, используя данные jsonUser.
*/
func CallDbUserFabric(jsonUser UserWithPassword) (instance *DbUser, err error) {
	instance = &DbUser{}
	instance.SetLogin(jsonUser.GetLogin())
	err = instance.SetHashedPassword(jsonUser.GetPassword())
	return
}
