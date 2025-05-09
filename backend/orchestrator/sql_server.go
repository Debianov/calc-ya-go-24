package main

import (
	"context"
	"database/sql"
	"github.com/Debianov/calc-ya-go-24/backend"
	"log"
)

type DbWrapper interface {
	InsertUser(user backend.UserWithHashedPassword) (lastId int64, err error)
	InsertExpr(expr backend.CommonExpression) (err error)
	SelectUser(login string) (user backend.UserWithHashedPassword, err error)
	SelectAllExprs(userOwnerId int64) (exprs []backend.ShortExpression, err error)
	SelectExpr(userOwnerId int64, exprId int) (expr backend.ShortExpression, err error)
	Flush() (err error)
	GetLastExprId() (int, error)
	Close() (err error)
}

type Db struct {
	ctx     context.Context
	innerDb *sql.DB
}

func (d *Db) InsertUser(user backend.UserWithHashedPassword) (lastId int64, err error) {
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

func (d *Db) InsertExpr(expr backend.CommonExpression) (err error) {
	var (
		query = `
	INSERT INTO exprs (exprId, ownerId, status, _result) values ($1, $2, $3, $4)
	`
	)
	_, err = d.innerDb.ExecContext(d.ctx, query, expr.GetId(), expr.GetOwnerId(), expr.GetStatus(), expr.GetResult())
	if err != nil {
		return
	}
	return
}

func (d *Db) SelectUser(login string) (resultedUser backend.UserWithHashedPassword, err error) {
	var (
		query = `
	SELECT id, login, password FROM users WHERE login=$1
	`
		userId             int64
		userLogin          string
		userHashedPassword string
	)
	err = d.innerDb.QueryRowContext(d.ctx, query, login).Scan(&userId, &userLogin, &userHashedPassword)
	resultedUser = backend.CallDbUserFabric(userId, userLogin, userHashedPassword)
	return
}

func (d *Db) SelectAllExprs(userOwnerId int64) (exprs []backend.ShortExpression, err error) {
	var (
		query = `
	SELECT exprId, status, _result FROM exprs WHERE ownerId=$1 
	`
		rows *sql.Rows
	)
	rows, err = d.innerDb.QueryContext(d.ctx, query, userOwnerId)
	defer rows.Close()
	for rows.Next() {
		var (
			expr   *backend.Expression
			id     int
			status backend.ExprStatus
			result int64
		)
		if err = rows.Scan(&id, &status, &result); err != nil {
			return
		}
		expr = backend.CallShortExpressionFabric(id, userOwnerId, status, result)
		exprs = append(exprs, expr)
	}
	return
}

func (d *Db) SelectExpr(userOwnerId int64, exprId int) (expr backend.ShortExpression, err error) {
	var (
		query = `
	SELECT status, _result FROM exprs WHERE ownerId=$1 AND exprId=$2 
	`
		status backend.ExprStatus
		result int64
	)
	err = d.innerDb.QueryRowContext(d.ctx, query, userOwnerId, exprId).Scan(&status, &result)
	expr = backend.CallShortExpressionFabric(exprId, userOwnerId, status, result)
	return
}

func (d *Db) GetLastExprId() (result int, err error) {
	var (
		query = `
	SELECT exprId FROM exprs ORDER BY exprId DESC LIMIT 1;
	`
	)
	err = d.innerDb.QueryRowContext(d.ctx, query).Scan(&result)
	return
}

func (d *Db) Flush() (err error) {
	var (
		query = `
	DELETE FROM users;
	DELETE FROM exprs;
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

func createBaseTables(ctx context.Context, db *sql.DB) (err error) {
	const usersTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		login TEXT UNIQUE,
		password TEXT
	);
	CREATE TABLE IF NOT EXISTS exprs(
	    exprId INTEGER PRIMARY KEY,
	    ownerId INTEGER,
	    status TEXT,
		_result INTEGER,
		FOREIGN KEY (ownerId) REFERENCES users (id)
	);
	`
	if _, err = db.ExecContext(ctx, usersTable); err != nil {
		return err
	}
	return
}
