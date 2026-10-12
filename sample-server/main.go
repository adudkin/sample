package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed static/*
var pages embed.FS

type entry struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS entries (id BIGSERIAL PRIMARY KEY, text TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "Database unavailable", 503)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /api/texts", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		defer r.Body.Close()
		var input struct {
			Text string `json:"text"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&input); err != nil || strings.TrimSpace(input.Text) == "" || len(input.Text) > 60000 {
			http.Error(w, "Введите текст длиной до 60000 байт", 400)
			return
		}
		var extra interface{}
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			http.Error(w, "Invalid JSON", 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		var item entry
		err := db.QueryRow(ctx, `INSERT INTO entries (text) VALUES ($1) RETURNING id, text, created_at`, input.Text).Scan(&item.ID, &item.Text, &item.CreatedAt)
		if err != nil {
			log.Print(err)
			http.Error(w, "Не удалось сохранить текст", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	})
	mux.HandleFunc("GET /api/texts", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		rows, err := db.Query(ctx, `SELECT id, text, created_at FROM entries ORDER BY id DESC`)
		if err != nil {
			log.Print(err)
			http.Error(w, "Не удалось загрузить записи", 500)
			return
		}
		defer rows.Close()
		items := []entry{}
		for rows.Next() {
			var item entry
			if err := rows.Scan(&item.ID, &item.Text, &item.CreatedAt); err != nil {
				http.Error(w, "Database error", 500)
				return
			}
			items = append(items, item)
		}
		if rows.Err() != nil {
			http.Error(w, "Database error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(items)
	})
	for route, file := range map[string]string{"GET /{$}": "static/index.html", "GET /admin": "static/admin.html"} {
		data, err := pages.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
		})
	}
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	log.Print("Listening on :8080")
	log.Fatal(server.ListenAndServe())
}
