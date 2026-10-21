package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cardColumns = `id, timestamp, request, offer, details, consideration, contact`

type postgresStore struct {
	db *pgxpool.Pool
}

func (store *postgresStore) initialize(ctx context.Context) error {
	// A fresh cards table replaces the old text-only application's data model.
	// Existing tables are never dropped implicitly at application startup.
	_, err := store.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS cards (
		id BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 1 INCREMENT BY 1) PRIMARY KEY,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
		request TEXT NOT NULL DEFAULT '',
		offer TEXT NOT NULL DEFAULT '',
		details TEXT NOT NULL DEFAULT '',
		consideration TEXT NOT NULL DEFAULT '',
		contact TEXT NOT NULL DEFAULT ''
	)`)
	return err
}

func (store *postgresStore) ping(ctx context.Context) error {
	return store.db.Ping(ctx)
}

func scanCard(row pgx.Row) (card, error) {
	var item card
	err := row.Scan(&item.ID, &item.Timestamp, &item.Request, &item.Offer,
		&item.Details, &item.Consideration, &item.Contact)
	return item, err
}

func (store *postgresStore) create(ctx context.Context, input cardFields) (card, error) {
	return scanCard(store.db.QueryRow(ctx, `INSERT INTO cards
		(request, offer, details, consideration, contact) VALUES ($1, $2, $3, $4, $5)
		RETURNING `+cardColumns, input.Request, input.Offer, input.Details,
		input.Consideration, input.Contact))
}

func (store *postgresStore) find(ctx context.Context, id int64) (card, error) {
	item, err := scanCard(store.db.QueryRow(ctx, `SELECT `+cardColumns+` FROM cards WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return card{}, errNotFound
	}
	return item, err
}

func (store *postgresStore) list(ctx context.Context) ([]card, error) {
	rows, err := store.db.Query(ctx, `SELECT `+cardColumns+` FROM cards ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]card, 0)
	for rows.Next() {
		item, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
