package main

import (
	"context"
	"database/sql"
	"log"
)

type DbWrapper interface {
	InsertUser(user UserWithHashedPassword) (lastId int64, err error)
	SelectUser(login string) (user UserWithHashedPassword, err error)
	Flush() (err error)
	Close() (err error)
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

func (d *Db) SelectUser(login string) (resultedUser UserWithHashedPassword, err error) {
	var (
		query = `
	SELECT id, login, password FROM users WHERE login=$1
	`
	)
	var user = &DbUser{}
	err = d.innerDb.QueryRowContext(d.ctx, query, login).Scan(&user.id, &user.login, &user.hashedPassword)
	resultedUser = UserWithHashedPassword(user)
	return
}

func (d *Db) Flush() (err error) {
	var (
		query = `
	DELETE FROM users;
	`
	)
	_, err = d.innerDb.ExecContext(d.ctx, query)
	return
}

func (d *Db) Close() (err error) {
	return d.innerDb.Close()
}

func CallDbFabric() *Db {
	var (
		innerDb = GetDefaultSqlServer()
		ctx     = context.TODO()
		err     error
	)
	err = innerDb.PingContext(ctx)
	if err != nil {
		log.Panic(err)
	}
	err = createBaseTables(ctx, innerDb)
	if err != nil {
		log.Panic(err)
	}
	return &Db{ctx: ctx, innerDb: innerDb}
}

func CallTestDbFabric() (*Db, error) {
	var (
		innerDb = GetTestSqlServer()
		ctx     = context.TODO()
		err     error
	)
	err = innerDb.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	err = createBaseTables(ctx, innerDb)
	if err != nil {
		return nil, err
	}
	return &Db{ctx: ctx, innerDb: innerDb}, nil
}

func createBaseTables(ctx context.Context, db *sql.DB) (err error) {
	const usersTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		login TEXT UNIQUE,
		password TEXT
	);`
	if _, err = db.ExecContext(ctx, usersTable); err != nil {
		return err
	}
	return
}
