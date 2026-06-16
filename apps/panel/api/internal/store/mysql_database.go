package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func ensureDatabaseExists(dsn string) error {
	config, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return err
	}
	databaseName := config.DBName
	if databaseName == "" {
		return nil
	}
	config.DBName = ""
	rootDB, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return err
	}
	defer rootDB.Close()
	var pingErr error
	for attempt := 0; attempt < 30; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		pingErr = rootDB.PingContext(ctx)
		cancel()
		if pingErr == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		return pingErr
	}
	quotedName := "`" + strings.ReplaceAll(databaseName, "`", "``") + "`"
	_, err = rootDB.Exec(
		"CREATE DATABASE IF NOT EXISTS " + quotedName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	)
	return err
}
