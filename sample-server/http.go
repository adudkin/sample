package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"time"
)

// One MiB also permits the JSON-escaped representation of five 10,000-rune
// fields. Field limits are checked separately after decoding.
const maxBodyBytes = 1024 * 1024

type api struct {
	store cardStore
}

type apiError struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func newHandler(store cardStore) http.Handler {
	api := &api{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("POST /api/cards", api.create)
	mux.HandleFunc("GET /api/cards", api.list)
	mux.HandleFunc("GET /api/cards/{id}", api.find)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("database operation: %v", err)
	writeJSON(w, http.StatusInternalServerError, apiError{Error: "Не удалось выполнить запрос. Попробуйте позже."})
}

func (api *api) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := api.store.ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "База данных недоступна."})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte("ok"))
}

func decodeFields(r io.Reader) (cardFields, error) {
	decoder := json.NewDecoder(r)
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return cardFields{}, err
	}
	if raw == nil {
		return cardFields{}, errors.New("expected a JSON object")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return cardFields{}, errors.New("unexpected content after JSON object")
	}
	var input cardFields
	fields := map[string]*string{
		"request": &input.Request, "offer": &input.Offer, "details": &input.Details,
		"consideration": &input.Consideration, "contact": &input.Contact,
	}
	for name, value := range raw {
		target, known := fields[name]
		if !known || len(value) == 0 || value[0] != '"' {
			return cardFields{}, errors.New("unknown field or non-string value")
		}
		if err := json.Unmarshal(value, target); err != nil {
			return cardFields{}, err
		}
	}
	return input, nil
}

func (api *api) create(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeJSON(w, http.StatusUnsupportedMediaType, apiError{Error: "Требуется Content-Type: application/json."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()
	input, err := decodeFields(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "Некорректный JSON: передайте только текстовые поля формы."})
		return
	}
	if fields := input.validate(); len(fields) != 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "Проверьте заполнение полей.", Fields: fields})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := api.store.create(ctx, input)
	if err != nil {
		serverError(w, err)
		return
	}
	w.Header().Set("Location", "/api/cards/"+strconv.FormatInt(item.ID, 10))
	writeJSON(w, http.StatusCreated, item)
}

func (api *api) find(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusNotFound, apiError{Error: "Не найдено"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	item, err := api.store.find(ctx, id)
	if errors.Is(err, errNotFound) {
		writeJSON(w, http.StatusNotFound, apiError{Error: "Не найдено"})
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (api *api) list(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := api.store.list(ctx)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
