package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type appCreateUserRequest struct {
	AuthLogin    string `json:"auth_login"`
	AuthPassword string `json:"auth_pass"`
	UserLogin    string `json:"user_login"`
	UserPassword string `json:"user_pass"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	var body []byte
	if body, err = ioutil.ReadAll(r.Body); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	msu.Info(ctx,
		zap.String("request", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")),
		zap.Any("body", string(body)))

	var request appCreateUserRequest
	if err = json.Unmarshal(body, &request); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if request.AuthLogin != "root" || request.AuthPassword != "azaza" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	count := 0
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE name = $1`, request.UserLogin).Scan(&count); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if count != 0 {
		w.WriteHeader(http.StatusConflict)
		return
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (name, password) VALUES ($1, $2)`,
		request.UserLogin,
		fmt.Sprintf("%x", md5.Sum([]byte(request.UserPassword)))); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

	login := r.URL.Query().Get("login")
	password := r.URL.Query().Get("password")

	if login == "" || password == "" {
		msu.Error(ctx,
			errors.New("login or password not set"),
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var rows *sql.Rows
	if rows, err = db.QueryContext(ctx,
		`SELECT id FROM users WHERE name = $1 AND password = $2`,
		login,
		fmt.Sprintf("%x", md5.Sum([]byte(password)))); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	id := 0
	if rows.Next() {
		if err = rows.Scan(&id); err != nil {
			msu.Error(ctx,
				err,
				zap.Any("uri", r.RequestURI),
				zap.Any("query", r.URL.Query()),
				zap.Any("AuthHeader", r.Header.Get("Authorization")))
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		msu.Error(ctx,
			errors.New("invalid login or password"),
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rows.Close()

	token := generateUUID()

	if _, err = db.ExecContext(ctx,
		`UPDATE users SET app_token = $1 WHERE id = $2`,
		token,
		id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, token)
}
