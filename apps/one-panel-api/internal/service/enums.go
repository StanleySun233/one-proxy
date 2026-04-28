package service

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (c *ControlPlane) LoadEnums() error {
	items, err := c.store.ListFieldEnums()
	if err != nil {
		return err
	}
	c.enumsByField = make(map[string]map[string]domain.FieldEnum)
	for _, item := range items {
		if c.enumsByField[item.Field] == nil {
			c.enumsByField[item.Field] = make(map[string]domain.FieldEnum)
		}
		c.enumsByField[item.Field][item.Value] = item
	}
	return nil
}

func (c *ControlPlane) ListFieldEnums() ([]domain.FieldEnum, error) {
	return c.store.ListFieldEnums()
}

func (c *ControlPlane) ListFieldEnumsByField(field string) ([]domain.FieldEnum, error) {
	return c.store.ListFieldEnumsByField(field)
}

func (c *ControlPlane) enumValues(field string) map[string]domain.FieldEnum {
	if c.enumsByField == nil {
		return nil
	}
	return c.enumsByField[field]
}

func (c *ControlPlane) isValidEnum(field, value string) bool {
	m := c.enumValues(field)
	if m == nil {
		return true
	}
	_, ok := m[value]
	return ok
}
