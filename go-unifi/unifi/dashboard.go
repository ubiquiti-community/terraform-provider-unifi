// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListDashboard(ctx context.Context, site string) ([]Dashboard, error) {
	return c.listDashboard(ctx, site)
}

func (c *ApiClient) GetDashboard(ctx context.Context, site, id string) (*Dashboard, error) {
	return c.getDashboard(ctx, site, id)
}

func (c *ApiClient) DeleteDashboard(ctx context.Context, site, id string) error {
	return c.deleteDashboard(ctx, site, id)
}

func (c *ApiClient) CreateDashboard(
	ctx context.Context,
	site string,
	d *Dashboard,
) (*Dashboard, error) {
	return c.createDashboard(ctx, site, d)
}

func (c *ApiClient) UpdateDashboard(
	ctx context.Context,
	site string,
	d *Dashboard,
) (*Dashboard, error) {
	return c.updateDashboard(ctx, site, d)
}
