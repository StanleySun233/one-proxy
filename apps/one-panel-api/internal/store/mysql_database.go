package store

import (
	"database/sql"
	"strings"

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
	if err := rootDB.Ping(); err != nil {
		return err
	}
	quotedName := "`" + strings.ReplaceAll(databaseName, "`", "``") + "`"
	_, err = rootDB.Exec(
		"CREATE DATABASE IF NOT EXISTS " + quotedName + " CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	)
	return err
}
