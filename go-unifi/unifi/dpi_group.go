// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListDpiGroup(ctx context.Context, site string) ([]DpiGroup, error) {
	return c.listDpiGroup(ctx, site)
}

func (c *ApiClient) GetDpiGroup(ctx context.Context, site, id string) (*DpiGroup, error) {
	return c.getDpiGroup(ctx, site, id)
}

func (c *ApiClient) DeleteDpiGroup(ctx context.Context, site, id string) error {
	return c.deleteDpiGroup(ctx, site, id)
}

func (c *ApiClient) CreateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error) {
	return c.createDpiGroup(ctx, site, d)
}

func (c *ApiClient) UpdateDpiGroup(ctx context.Context, site string, d *DpiGroup) (*DpiGroup, error) {
	return c.updateDpiGroup(ctx, site, d)
}
