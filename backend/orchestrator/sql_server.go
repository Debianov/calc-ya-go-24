package main

import (
	"context"
	"database/sql"
)

func CallDbFabric() (*Db, error) {
	var (
		innerDb = GetDefaultSqlServer()
		ctx     = context.TODO()
		err     error
	)
	err = innerDb.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return &Db{ctx: ctx, innerDb: innerDb}, nil
}

type Db struct {
	ctx     context.Context
	innerDb *sql.DB
}

func (d *Db) InsertUser(user UserWithHashedPassword) (lastId int64, err error) {
	var (
		query = `
	INSERT INTO users (login, password) values ($1, $2)
	`
		result sql.Result
	)
	result, err = d.innerDb.ExecContext(d.ctx, query, user.GetLogin(), user.GetHashedPassword())
	if err != nil {
		return
	}
	lastId, err = result.LastInsertId()
	return
}

func (d *Db) SelectUser(login string) (user DbUser, err error) {
	var (
		query = `
	SELECT id, login, password FROM users WHERE login=$1
	`
	)
	err = d.innerDb.QueryRowContext(d.ctx, query, login).Scan(&user.id, &user.login, &user.hashedPassword)
	return
}

func (d *Db) Close() (err error) {
	return d.innerDb.Close()
}

type UserWithHashedPassword interface {
	CommonUser
	GetHashedPassword() string
	SetHashedPassword(salt string) (err error)
}

/*
DbUser -- структура для внутреннего использования (контур db - backend).
*/
type DbUser struct {
	id             int64
	login          string
	hashedPassword string
	hashMan        HashMan
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
