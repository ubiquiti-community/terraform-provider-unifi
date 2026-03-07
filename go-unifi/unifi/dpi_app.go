// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListDpiApp(ctx context.Context, site string) ([]DpiApp, error) {
	return c.listDpiApp(ctx, site)
}

func (c *ApiClient) GetDpiApp(ctx context.Context, site, id string) (*DpiApp, error) {
	return c.getDpiApp(ctx, site, id)
}

func (c *ApiClient) DeleteDpiApp(ctx context.Context, site, id string) error {
	return c.deleteDpiApp(ctx, site, id)
}

func (c *ApiClient) CreateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error) {
	return c.createDpiApp(ctx, site, d)
}

func (c *ApiClient) UpdateDpiApp(ctx context.Context, site string, d *DpiApp) (*DpiApp, error) {
	return c.updateDpiApp(ctx, site, d)
}
