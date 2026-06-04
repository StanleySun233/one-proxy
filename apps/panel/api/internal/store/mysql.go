package store

import (
	"context"
	"database/sql"
	"os"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLStore struct {
	gormDB                 *gorm.DB
	db                     *sql.DB
	bootstrapAdminPassword string
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	if err := ensureDatabaseExists(dsn); err != nil {
		return nil, err
	}
	gormDB, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	store := &MySQLStore{
		gormDB: gormDB,
		db:     db,
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
	schemaFiles, err := resolveSchemaFiles()
	if err != nil {
		return err
	}
	for _, schemaPath := range schemaFiles {
		schemaBytes, err := os.ReadFile(schemaPath)
		if err != nil {
			return err
		}
		statements := splitSQLStatements(string(schemaBytes))
		for _, statement := range statements {
			if _, err := s.db.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	if err := s.gormDB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return err
	}
	if err := s.bootstrapAdmin(ctx); err != nil {
		return err
	}
	if err := s.ensureBootstrapTokenMetadataColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureNodeAccessPathProtocolColumns(ctx); err != nil {
		return err
	}
	if err := s.ensureAccountRoleModel(ctx); err != nil {
		return err
	}
	if err := s.bootstrapConfig(ctx); err != nil {
		return err
	}
	return nil
}
