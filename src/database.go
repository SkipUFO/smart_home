package main

import (
	"context"
	"database/sql"
	"os"

	"go.uber.org/zap"
)

func initializeDB(c context.Context, path string) error {
	ctx := c
	msu.Info(ctx, zap.Any("initializeDB", path))
	// If file exists than ok - exit without errors
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	msu.Info(ctx, zap.Any("database", "creating new"))
	if _, err := os.Create(path); err != nil {
		return err
	}

	// Initialize database
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		msu.Fatal(context.Background(), err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS users (
		id             INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		name           TEXT    NOT NULL,
		password       TEXT NOT NULL,
		yandex_code    TEXT,
		yandex_token   TEXT, 
		app_token      TEXT)`); err != nil {
		os.Remove(path)
		return err
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS controllers (
		id             INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		user_id        INTEGER NOT NULL, 
		name           TEXT    NOT NULL,
		password       TEXT NOT NULL,
		uri            TEXT, 
		FOREIGN KEY(user_id) REFERENCES users(id))`); err != nil {
		os.Remove(path)
		return err
	}

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS auth_requests (
		id TEXT NOT NULL, 
		request TEXT NOT NULL, 
		dt TEXT NOT NULL)`); err != nil {
		os.Remove(path)
		return err
	}

	msu.Info(ctx, zap.Any("database", "created"))
	return nil
}
