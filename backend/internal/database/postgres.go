package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultPingTimeout     = 5 * time.Second
)

type Config struct {
	DatabaseURL     string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

func Open(ctx context.Context, cfg Config) (*bun.DB, error) {
	dsn := strings.TrimSpace(cfg.DatabaseURL)
	if dsn == "" {
		return nil, errors.New("database url is required")
	}

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqlDB.SetMaxOpenConns(valueOrDefault(cfg.MaxOpenConns, defaultMaxOpenConns))
	sqlDB.SetMaxIdleConns(valueOrDefault(cfg.MaxIdleConns, defaultMaxIdleConns))
	sqlDB.SetConnMaxLifetime(durationOrDefault(cfg.ConnMaxLifetime, defaultConnMaxLifetime))

	db := bun.NewDB(sqlDB, pgdialect.New())

	pingTimeout := durationOrDefault(cfg.PingTimeout, defaultPingTimeout)
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

func Close(db *bun.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

func valueOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func durationOrDefault(value time.Duration, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}
