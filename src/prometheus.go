package main

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/ms-ural/airport/core/logger.git"
	"go.uber.org/zap"
)

func generateUUID() string {
	u := uuid.New()
	return u.String()
}

func prometheusHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))
		w.Header().Set("X-Correlation-ID", r.Header.Get("X-Correlation-ID"))

		traceID := r.Header.Get("X-Request-ID")
		if traceID == "" {
			traceID = generateUUID()
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, logger.RequestURIContextKey, r.RequestURI)
		ctx = context.WithValue(ctx, logger.RemoteAddrContextKey, r.RemoteAddr)
		ctx = context.WithValue(ctx, logger.TraceIDContextKey, traceID)
		ctx = context.WithValue(ctx, logger.SessionIDContextKey, r.Header.Get("X-Correlation-ID"))

		msu.Info(ctx, zap.Any("URL", r.URL.Path), zap.Any("method", r.Method))
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			getDurations.WithLabelValues(r.URL.Path).Observe(v)
		}))
		defer timer.ObserveDuration()

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
