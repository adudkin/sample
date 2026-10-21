package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	items []card
	err   error
}

func (store *fakeStore) ping(context.Context) error { return store.err }
func (store *fakeStore) create(_ context.Context, input cardFields) (card, error) {
	if store.err != nil {
		return card{}, store.err
	}
	item := card{ID: int64(len(store.items) + 1), Timestamp: time.Now().UTC(), cardFields: input}
	store.items = append(store.items, item)
	return item, nil
}
func (store *fakeStore) find(_ context.Context, id int64) (card, error) {
	if store.err != nil {
		return card{}, store.err
	}
	for _, item := range store.items {
		if item.ID == id {
			return item, nil
		}
	}
	return card{}, errNotFound
}
func (store *fakeStore) list(context.Context) ([]card, error) {
	return store.items, store.err
}

func perform(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestCreateReadAndListCard(t *testing.T) {
	store := &fakeStore{items: []card{}}
	handler := newHandler(store)
	response := perform(handler, "POST", "/api/cards", `{"request":"Нужно 2 стула\nдо пятницы","offer":"<script>alert(1)</script>","details":"Нижний Новгород","consideration":"Доставка?","contact":"Анна"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Location") != "/api/cards/1" {
		t.Fatalf("missing created-card location: %v", response.Header())
	}
	var created card
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Timestamp.IsZero() || created.Request != "Нужно 2 стула\nдо пятницы" || created.Offer != "<script>alert(1)</script>" {
		t.Fatalf("unexpected card: %#v", created)
	}
	response = perform(handler, "GET", "/api/cards/1", "")
	if response.Code != http.StatusOK {
		t.Fatalf("read: %d %s", response.Code, response.Body.String())
	}
	var found card
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil || found != created {
		t.Fatalf("read mismatch: %#v, %v", found, err)
	}
	response = perform(handler, "GET", "/api/cards", "")
	var listed []card
	if err := json.Unmarshal(response.Body.Bytes(), &listed); response.Code != http.StatusOK || err != nil || len(listed) != 1 || listed[0] != created {
		t.Fatalf("list: %d %s, %v", response.Code, response.Body.String(), err)
	}
}

func TestInvalidInputNeverReachesStorage(t *testing.T) {
	inputs := []string{
		`{}`, `{"request":"   \n"}`, `{"request":42}`, `{"request":null}`,
		`{"request":"ok","offer":null}`, `{"request":"ok","id":1}`,
		`{"request":"ok"} {}`, `{"request":"ok"} garbage`, `null`, `[]`, `{"request":`,
		`{"request":"` + strings.Repeat("я", maxFieldLength+1) + `"}`,
		`{"request":"ok","offer":"` + strings.Repeat("a", maxFieldLength+1) + `"}`,
		`{"request":"` + strings.Repeat("a", maxBodyBytes+1) + `"}`,
	}
	for _, body := range inputs {
		store := &fakeStore{items: []card{}}
		response := perform(newHandler(store), "POST", "/api/cards", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("body %.70q: got %d %s", body, response.Code, response.Body.String())
		}
		if len(store.items) != 0 {
			t.Errorf("invalid input was saved: %.70q", body)
		}
	}
}

func TestValidationErrorNamesField(t *testing.T) {
	response := perform(newHandler(&fakeStore{}), "POST", "/api/cards", `{"offer":"Помощь"}`)
	var result apiError
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Fields["request"] == "" {
		t.Fatalf("missing inline request error: %s, %v", response.Body.String(), err)
	}
}

func TestUnicodeLimitAndOptionalFields(t *testing.T) {
	store := &fakeStore{items: []card{}}
	body := `{"request":"` + strings.Repeat("я", maxFieldLength) + `"}`
	response := perform(newHandler(store), "POST", "/api/cards", body)
	if response.Code != http.StatusCreated || len(store.items) != 1 || store.items[0].Contact != "" {
		t.Fatalf("valid unicode limit and omitted optional fields: %d", response.Code)
	}
}

func TestUnknownIDs(t *testing.T) {
	handler := newHandler(&fakeStore{})
	for _, id := range []string{"1", "0", "-1", "abc", "9223372036854775808"} {
		response := perform(handler, "GET", "/api/cards/"+id, "")
		var result apiError
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 404 || result.Error != "Не найдено" {
			t.Errorf("id %s: %d %s", id, response.Code, response.Body.String())
		}
	}
}

func TestDatabaseFailures(t *testing.T) {
	handler := newHandler(&fakeStore{err: errors.New("database offline")})
	for _, test := range []struct {
		method, path, body string
		status             int
	}{
		{"GET", "/healthz", "", http.StatusServiceUnavailable},
		{"GET", "/api/cards", "", http.StatusInternalServerError},
		{"GET", "/api/cards/1", "", http.StatusInternalServerError},
		{"POST", "/api/cards", `{"request":"Нужно"}`, http.StatusInternalServerError},
	} {
		response := perform(handler, test.method, test.path, test.body)
		if response.Code != test.status || strings.Contains(response.Body.String(), "database offline") {
			t.Errorf("%s %s: %d %s", test.method, test.path, response.Code, response.Body.String())
		}
	}
}

func TestHealthEmptyListAndContentType(t *testing.T) {
	handler := newHandler(&fakeStore{items: []card{}})
	response := perform(handler, "GET", "/api/cards", "")
	if response.Code != 200 || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", response.Code, response.Body.String())
	}
	response = perform(handler, "GET", "/healthz", "")
	if response.Code != 200 || response.Body.String() != "ok" {
		t.Fatalf("health: %d %s", response.Code, response.Body.String())
	}
	request := httptest.NewRequest("POST", "/api/cards", strings.NewReader(`{"request":"ok"}`))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("missing content type: %d", response.Code)
	}
}
