package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"net/http"

	"github.com/gorilla/mux"
	"gitlab.com/ms-ural/airport/core/logger.git"
	"golang.org/x/crypto/acme/autocert"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"

	"log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Product - event.module in ECS
	Product string = "bsh"
	// Component - event.provider in ECS
	Component string = "backend"
)

var (
	msu               *logger.MsuLogger
	databaseErrors    prometheus.Counter
	getDurations      *prometheus.HistogramVec
	httpsEnabled      = true
	timeout           = 15
	databaseDirectory = "/tmp"
	db                *sql.DB

	debug = true
)

func main() {
	var err error

	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Development: false,
	}

	zaplogger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))

	if err != nil {
		log.Fatal(err)
	}
	defer zaplogger.Sync()

	msu = logger.NewMsuLogger(zaplogger, Product, Component)

	/** PROMETHEUS */
	/* Database errors counter */
	databaseErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "database_errors",
		})
	prometheus.MustRegister(databaseErrors)
	getDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "get_durations_seconds",
		Help:    "duration histogram",
		Buckets: []float64{0.05, 0.1, 0.25, 0.75, 1, 2},
	},
		[]string{"api"})
	prometheus.MustRegister(getDurations)

	corsOpts := cors.New(cors.Options{
		AllowedOrigins: []string{"*"}, //you service is available and allowed for this base url
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodOptions,
		},

		AllowedHeaders: []string{
			"*", //or you can your header key values which you are using in your application

		},
		ExposedHeaders: []string{
			"*",
		},
	})

	if _, ok := os.LookupEnv("HTTPS_DISABLED"); ok {
		httpsEnabled = false
	}

	if err := initializeDB(context.Background(), databaseDirectory+"/users.db"); err != nil {
		msu.Fatal(context.Background(), err)
	}
	db, err = sql.Open("sqlite3", databaseDirectory+"/users.db")
	if err != nil {
		msu.Fatal(context.Background(), err)
	}
	defer db.Close()

	if httpsEnabled {
		dir := "/opt/certs"
		hostPolicy := func(ctx context.Context, host string) error {
			return nil
		}
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: hostPolicy,
			Cache:      autocert.DirCache(dir),
			Email:      "info@msural.ru",
		}

		server := &http.Server{
			Addr:      ":8443",
			Handler:   prometheusHandler(corsOpts.Handler(handlers())),
			TLSConfig: certManager.TLSConfig(),
		}

		go func(s *http.Server) {
			err := http.ListenAndServe(":8080", certManager.HTTPHandler(nil))
			if err != nil {
				if e := s.Shutdown(context.Background()); e != nil {
					msu.Fatal(context.Background(), e)
				}
				msu.Fatal(context.Background(), err)
			}
		}(server)

		err = server.ListenAndServeTLS("", "")
	} else {
		err = http.ListenAndServe(":8080", prometheusHandler(corsOpts.Handler(handlers())))
		//err = http.ListenAndServe(":8080", corsOpts.Handler(handlers()))
	}

	if err != nil {
		msu.Fatal(context.Background(), err)
	}
}

func handlers() http.Handler {
	r := mux.NewRouter()

	// Yandex API
	r.HandleFunc("/auth/authorize", authorize)
	r.HandleFunc("/auth/token", token)
	r.HandleFunc("/api/v1.0/user/unlink", unlink).Methods(http.MethodPost)
	r.HandleFunc("/api/v1.0/user/devices", devices).Methods(http.MethodGet)
	r.HandleFunc("/api/v1.0/user/devices/action", action).Methods(http.MethodPost)
	r.HandleFunc("/api/v1.0/user/devices/query", query).Methods(http.MethodPost)
	// Auth API (For Alisa)
	r.HandleFunc("/auth/login", login).Methods(http.MethodPost)
	// Install App
	// Users
	r.HandleFunc("/users/register", createUser).Methods(http.MethodPost)
	r.HandleFunc("/users/auth", loginUser).Methods(http.MethodPost)
	// Controllers
	r.HandleFunc("/controllers", getControllers).Methods(http.MethodGet)
	r.HandleFunc("/controllers/{id}", getController).Methods(http.MethodGet)
	r.HandleFunc("/controllers", createController).Methods(http.MethodPost)
	r.HandleFunc("/controllers/{id}", updateController).Methods(http.MethodPut)
	r.HandleFunc("/controllers/{id}", deleteController).Methods(http.MethodDelete)
	// PROMETHEUS
	r.Handle("/metrics", promhttp.Handler())

	return r
}

func authorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	requestID := generateUUID()

	if _, err := db.ExecContext(ctx,
		`INSERT INTO auth_requests (id, request, dt) VALUES ($1, $2, $3)`,
		requestID, r.URL.RawQuery, time.Now().Format(time.RFC3339)); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	msu.Info(ctx, zap.Any("query", r.URL.Query()))

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/text;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Request-Id", requestID)
	fmt.Fprint(w, string(`<!DOCTYPE html>
	<html lang="ru">
	<head>
		<meta charset="UTF-8">
		<title>Login</title>
	</head>
	<body><form action="/auth/login" method="post">
		<div class="imgcontainer">
		<img src="img_avatar2.png" class="avatar">
		</div>
	
		<div class="container">
		<label for="uname"><b>Username</b></label>
		<input type="text" placeholder="Enter Username" name="username" required>
		<br>
		<label for="psw"><b>Password</b></label>
		<input type="password" placeholder="Enter Password" name="password" required>
		<br>	
		<button type="submit">Login</button>
		</div>
		<input type="hidden" name="rid" value="`+requestID+`" required>
			
	</form>
	</body>
</html>`))

	// <div class="container" style="background-color:#f1f1f1">
	// <button type="button" class="cancelbtn">Cancel</button>
	//	<span class="psw">Forgot <a href="#">password?</a></span>
	//	</div>

	//w.Header().Set("Location", "https://social.yandex.net/broker/redirect?"+r.URL.RawQuery+"&code="+generateUUID())
	//w.WriteHeader(http.StatusMovedPermanently)
}

func login(w http.ResponseWriter, r *http.Request) {
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

	username := ""
	password := ""
	request := ""
	// credentials from body
	for _, val := range strings.Split(string(body), "&") {
		if strings.HasPrefix(val, "username") {
			temp := strings.Split(val, "=")
			if len(temp) == 2 {
				username = temp[len(temp)-1]
			}
			continue
		}

		if strings.HasPrefix(val, "password") {
			temp := strings.Split(val, "=")
			if len(temp) == 2 {
				password = temp[len(temp)-1]
			}
			continue
		}

		if strings.HasPrefix(val, "rid") {
			temp := strings.Split(val, "=")
			if len(temp) == 2 {
				request = temp[len(temp)-1]
			}
			continue
		}
	}
	//

	if username == "" || password == "" || request == "" {
		msu.Error(ctx,
			errors.New("username, password or requestId is empty"),
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check user exists
	var rows *sql.Rows
	if rows, err = db.QueryContext(ctx,
		`SELECT id FROM users WHERE name = $1 AND password = $2`,
		username,
		fmt.Sprintf("%x", md5.Sum([]byte(password)))); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
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
				zap.Any("AuthHeader", r.Header.Get("Authorization")),
				zap.Any("body", string(body)))
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		msu.Error(ctx,
			errors.New("invalid login or password"),
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	rows.Close()
	//

	// get r.URL.RawQuery for yandex response
	urlRawQuery := ""
	if err = db.QueryRowContext(ctx, `SELECT request FROM auth_requests WHERE id = $1`, request).Scan(&urlRawQuery); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//

	code := generateUUID()
	// set code to users
	if _, err = db.ExecContext(ctx, `UPDATE users SET yandex_code = $1 WHERE name = $2`, code, username); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//

	// remove request from auth_requests
	if _, err = db.ExecContext(ctx, `DELETE FROM auth_requests WHERE id = $1`, request); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")),
			zap.Any("body", string(body)))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//

	w.Header().Set("Location", "https://social.yandex.net/broker/redirect?"+urlRawQuery+"&code="+code)
	w.WriteHeader(http.StatusMovedPermanently)

}

func token(w http.ResponseWriter, r *http.Request) {
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

	code := ""
	// credentials from body
	for _, val := range strings.Split(string(body), "&") {
		if strings.HasPrefix(val, "code") {
			temp := strings.Split(val, "=")
			if len(temp) == 2 {
				code = temp[len(temp)-1]
			}
			continue
		}
	}
	//

	var commandTag sql.Result
	token := generateUUID()
	// select and update token from code
	if commandTag, err = db.ExecContext(ctx, `UPDATE users SET yandex_token = $1 WHERE yandex_code = $2`, token, code); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if i, err := commandTag.RowsAffected(); err != nil {
		if i == 0 {
			msu.Error(ctx,
				err,
				zap.Any("uri", r.RequestURI),
				zap.Any("query", r.URL.Query()),
				zap.Any("AuthHeader", r.Header.Get("Authorization")))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	//

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w,
		string(`{
			"access_token":"`+token+`",
			"token_type":"Bearer",
			"expires_in":3600,
			"refresh_token":"`+token+`"
		}`),
	)
}

func unlink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

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

	msu.Info(ctx,
		zap.String("request", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")))

	var commandTag sql.Result
	// select and update token from code
	if commandTag, err = db.ExecContext(ctx, `UPDATE users SET yandex_token = null WHERE yandex_code = $1`, token); err != nil {
		msu.Error(ctx,
			err,
			zap.Any("uri", r.RequestURI),
			zap.Any("query", r.URL.Query()),
			zap.Any("AuthHeader", r.Header.Get("Authorization")))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if i, err := commandTag.RowsAffected(); err != nil {
		if i == 0 {
			msu.Error(ctx,
				err,
				zap.Any("uri", r.RequestURI),
				zap.Any("query", r.URL.Query()),
				zap.Any("AuthHeader", r.Header.Get("Authorization")))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	//

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	fmt.Fprint(w,
		string(`{
			"request_id":"`+r.Header.Get("X-Request-Id")+`"
		}`),
	)
}

func devices(w http.ResponseWriter, r *http.Request) {
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

	msu.Info(ctx,
		zap.String("request", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")))

	result, err := getUserDevices(ctx, r.Header.Get("X-Request-Id"), token)
	if err != nil {
		if err.Error() == "account_linking_error" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		msu.Error(ctx, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	msu.Info(ctx,
		zap.String("response", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")),
		zap.String("body", result))

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, result)
}

func action(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var token string
	tokenInfo := strings.Split(r.Header.Get("Authorization"), " ")
	if len(tokenInfo) == 2 {
		token = tokenInfo[1]
	} else {
		msu.Error(ctx,
			errors.New("invalid AuthHeader len"),
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

	msu.Info(ctx,
		zap.String("request", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")),
		zap.Any("body", body))

	result, err := deviceAction(ctx, r.Header.Get("X-Request-Id"), token, body)

	if err != nil {
		if err.Error() == "account_linking_error" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		msu.Error(ctx, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	msu.Info(ctx,
		zap.String("response", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")),
		zap.Any("body", result))

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, result)
}

func query(w http.ResponseWriter, r *http.Request) {
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
		zap.Any("body", body))

	result, err := deviceQuery(ctx, r.Header.Get("X-Request-Id"), token, body)

	if err != nil {
		if err.Error() == "account_linking_error" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		msu.Error(ctx, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	msu.Info(ctx,
		zap.String("response", "yandex"),
		zap.Any("uri", r.RequestURI),
		zap.Any("query", r.URL.Query()),
		zap.Any("AuthHeader", r.Header.Get("Authorization")),
		zap.Any("body", result))

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, result)
}
