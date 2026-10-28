package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arbeiter/auth-platform/internal/auth"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if os.Getenv("APP_ENV") != "development" {
		u, parseErr := url.Parse(databaseURL)
		if parseErr != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || u.Query().Get("sslmode") != "verify-full" {
			logger.Error("DATABASE_URL must require sslmode=verify-full outside development")
			os.Exit(1)
		}
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	app, err := auth.New(db, logger)
	if err != nil {
		logger.Error("configuration", "error", err)
		os.Exit(1)
	}
	addr := env("LISTEN_ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	go func() {
		logger.Info("server listening", "addr", addr)
		if e := srv.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			logger.Error("http server", "error", e)
			os.Exit(1)
		}
	}()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if _, e := db.ExecContext(ctx, `DELETE FROM rate_limits WHERE window_start < now()-interval '1 day'`); e != nil {
				logger.Error("rate-limit cleanup failed", "error", e)
			}
			cancel()
		}
	}()
	stop, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()
	<-stop.Done()
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown", "error", err)
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
