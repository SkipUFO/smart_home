package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type controller struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	URI      string `json:"uri"`
}

func getControllers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var rows *sql.Rows
	var err error

	cnt := 0
	if err := db.QueryRowContext(ctx, `SELECT count(id) FROM users WHERE app_token = $1`, token).Scan(&cnt); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if cnt == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	controllers := make([]controller, 0)

	if rows, err = db.QueryContext(ctx,
		`SELECT id, name, password, uri FROM controllers WHERE user_id IN (SELECT id FROM users WHERE app_token = $1)`,
		token); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, password, uri string

		if err = rows.Scan(&id, &name, &password, &uri); err != nil {
			msu.Error(ctx,
				err,
				zap.Any("uri", r.RequestURI),
				zap.Any("query", r.URL.Query()),
				zap.Any("AuthHeader", r.Header.Get("Authorization")))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		controllers = append(controllers, controller{
			ID:       id,
			Name:     name,
			Password: password,
			URI:      uri,
		})
	}

	var result []byte

	if result, err = json.Marshal(controllers); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(result))
}

func getController(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	cnt := 0
	if err := db.QueryRowContext(ctx, `SELECT count(id) FROM users WHERE app_token = $1`, token).Scan(&cnt); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if cnt == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)

	id, err := toInt(vars, "id")
	if err != nil {
		msu.Warn(ctx, err, zap.Any("vars", vars))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var rows *sql.Rows

	cntl := controller{}

	if rows, err = db.QueryContext(ctx,
		`SELECT id, name, password, uri FROM controllers WHERE user_id IN (SELECT id FROM users WHERE app_token = $1) AND id = $2`,
		token, id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	if rows.Next() {
		var id int
		var name, password, uri string

		if err = rows.Scan(&id, &name, &password, &uri); err != nil {
			msu.Error(ctx,
				err,
				zap.Any("uri", r.RequestURI),
				zap.Any("query", r.URL.Query()),
				zap.Any("AuthHeader", r.Header.Get("Authorization")))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		cntl = controller{
			ID:       id,
			Name:     name,
			Password: password,
			URI:      uri,
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var result []byte

	if result, err = json.Marshal(cntl); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(result))
}

func createController(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user_id := 0
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE app_token = $1`, token).Scan(&user_id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

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

	cntl := controller{}
	if err = json.Unmarshal(body, &cntl); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err = db.ExecContext(ctx,
		`INSERT INTO controllers (user_id, name, password, uri) VALUES ($1, $2, $3, $4)`,
		user_id, cntl.Name, cntl.Password, cntl.URI); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func updateController(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user_id := 0
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE app_token = $1`, token).Scan(&user_id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)

	id, err := toInt(vars, "id")
	if err != nil {
		msu.Warn(ctx, err, zap.Any("vars", vars))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

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

	cntl := controller{}
	if err = json.Unmarshal(body, &cntl); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err = db.ExecContext(ctx,
		`UPDATE controllers SET name = $1, password = $2, uri = $3 WHERE id = $4`,
		cntl.Name, cntl.Password, cntl.URI, id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteController(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user_id := 0
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE app_token = $1`, token).Scan(&user_id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)

	id, err := toInt(vars, "id")
	if err != nil {
		msu.Warn(ctx, err, zap.Any("vars", vars))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err = db.ExecContext(ctx, `DELETE FROM controllers WHERE id = $1`, id); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
