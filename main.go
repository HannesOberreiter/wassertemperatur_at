package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/HannesOberreiter/wassertemperatur_at/internal"
)

func main() {
	db, err := internal.OpenDB(env("SQL_PATH", "db/wasser.db"))
	if err != nil {
		slog.Error("Datenbank konnte nicht geöffnet werden", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if len(os.Args) > 1 && os.Args[1] == "fetch" {
		fetchCtx, cancelFetch := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelFetch()
		if err := internal.UpdateTemperatures(fetchCtx, db); err != nil {
			slog.Error("Datenabruf fehlgeschlagen", "error", err)
			os.Exit(1)
		}
		return
	}

	go internal.StartCron(db, envDuration("CRON_INTERVAL", time.Hour))

	server := &http.Server{Addr: ":1323", Handler: internal.NewServer(db)}
	go func() {
		slog.Info("Server läuft", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Serverfehler", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
