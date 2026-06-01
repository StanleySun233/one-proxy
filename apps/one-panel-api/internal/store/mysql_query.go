package store

import (
	"context"
	"database/sql"
	"errors"
)

func (s *MySQLStore) exists(ctx context.Context, query string, args ...any) (bool, error) {
	var value int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&value)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}
