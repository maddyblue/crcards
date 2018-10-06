package main

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/acme/autocert"
)

const autocertPrefix = "autocert-"

type dbCache struct {
	*sql.DB
}

func NewDBCache(conn string) (autocert.Cache, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}
	return dbCache{db}, nil
}

func (db dbCache) Get(ctx context.Context, key string) ([]byte, error) {
	var data []byte
	if err := db.QueryRowContext(ctx, "SELECT s FROM config WHERE key = $1", autocertPrefix+key).Scan(&data); err == sql.ErrNoRows {
		return nil, autocert.ErrCacheMiss
	} else if err != nil {
		return nil, err
	}
	return data, nil
}

func (db dbCache) Put(ctx context.Context, key string, data []byte) error {
	_, err := db.ExecContext(ctx, "UPSERT INTO config (key, s) VALUES ($1, $2)", autocertPrefix+key, data)
	return err
}

func (db dbCache) Delete(ctx context.Context, key string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM config WHERE key = $1", autocertPrefix+key)
	return err
}
