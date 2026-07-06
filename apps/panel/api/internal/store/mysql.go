package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

type MySQLStore struct {
	db                     *sql.DB
	bunDB                  *bun.DB
	bootstrapAdminPassword string
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	if err := ensureDatabaseExists(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	store := &MySQLStore{
		db:    db,
		bunDB: bun.NewDB(db, mysqldialect.New()),
	}
	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *MySQLStore) BootstrapAdminPassword() string {
	return s.bootstrapAdminPassword
}

func (s *MySQLStore) init(ctx context.Context) error {
	if err := s.initSchema(ctx); err != nil {
		return err
	}
	if err := s.initRemoteSchema(ctx); err != nil {
		return err
	}
	if err := s.initDirectSchema(ctx); err != nil {
		return err
	}
	if err := s.db.PingContext(ctx); err != nil {
		return err
	}
	if err := s.bootstrapAdmin(ctx); err != nil {
		return err
	}
	if err := s.bootstrapConfig(ctx); err != nil {
		return err
	}
	return nil
}
