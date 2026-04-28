package store

import (
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) ListFieldEnums() ([]domain.FieldEnum, error) {
	rows, err := s.db.Query("SELECT id, field, value, name, meta FROM field_enum ORDER BY field, value")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.FieldEnum, 0)
	for rows.Next() {
		var item domain.FieldEnum
		if err := rows.Scan(&item.ID, &item.Field, &item.Value, &item.Name, &item.Meta); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) ListFieldEnumsByField(field string) ([]domain.FieldEnum, error) {
	rows, err := s.db.Query("SELECT id, field, value, name, meta FROM field_enum WHERE field = ? ORDER BY value", field)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.FieldEnum, 0)
	for rows.Next() {
		var item domain.FieldEnum
		if err := rows.Scan(&item.ID, &item.Field, &item.Value, &item.Name, &item.Meta); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}
