package main

type CommonUser interface {
	GetLogin() string
	SetLogin(string)
	GetId() int64
	SetId(int64)
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
