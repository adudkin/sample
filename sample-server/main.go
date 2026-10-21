package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// pgx reads the standard PGHOST, PGPORT, PGUSER, PGPASSWORD,
	// PGDATABASE and PGSSLMODE variables without interpolating a URL.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return errors.New("invalid PostgreSQL configuration")
	}
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	defer db.Close()
	store := &postgresStore{db: db}
	if err := store.initialize(ctx); err != nil {
		return err
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           newHandler(store),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stopped, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result := make(chan error, 1)
	go func() {
		log.Print("API listening on :8080")
		result <- server.ListenAndServe()
	}()
	select {
	case err := <-result:
		return err
	case <-stopped.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	}
}
