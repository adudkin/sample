package main

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const maxFieldLength = 10000

var errNotFound = errors.New("card not found")

// cardFields contains only values entered on the form. The database assigns
// the ID and creation timestamp; clients cannot submit either of them.
type cardFields struct {
	Request       string `json:"request"`
	Offer         string `json:"offer"`
	Details       string `json:"details"`
	Consideration string `json:"consideration"`
	Contact       string `json:"contact"`
}

type card struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	cardFields
}

func (fields cardFields) validate() map[string]string {
	errors := make(map[string]string)
	if strings.TrimSpace(fields.Request) == "" {
		errors["request"] = "Укажите, что вам нужно."
	}
	for name, value := range map[string]string{
		"request": fields.Request, "offer": fields.Offer, "details": fields.Details,
		"consideration": fields.Consideration, "contact": fields.Contact,
	} {
		if utf8.RuneCountInString(value) > maxFieldLength {
			errors[name] = "Не больше 10 000 символов."
		}
	}
	return errors
}

type cardStore interface {
	ping(context.Context) error
	create(context.Context, cardFields) (card, error)
	find(context.Context, int64) (card, error)
	list(context.Context) ([]card, error)
}
