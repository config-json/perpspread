package storage

import (
	"context"
	"fmt"

	"github.com/config-json/perpspread/internal/config"
	db "github.com/config-json/perpspread/internal/storage/db/generated"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	Queries *db.Queries
	pool    *pgxpool.Pool
}

func New(ctx context.Context) (*Storage, error) {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.Storage.User,
		config.Storage.Password,
		config.Storage.Host,
		config.Storage.Port,
		config.Storage.Database,
		config.Storage.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)

	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = config.Storage.MaxConns
	poolConfig.MinConns = config.Storage.MinConns
	poolConfig.MaxConnLifetime = config.Storage.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.Storage.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.Storage.HealthCheck

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)

	if err != nil {
		return nil, err
	}

	return &Storage{
		Queries: db.New(pool),
		pool:    pool,
	}, nil
}

func (s *Storage) Close() {
	s.pool.Close()
}
