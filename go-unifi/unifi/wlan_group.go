// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListWLANGroup(ctx context.Context, site string) ([]WLANGroup, error) {
	return c.listWLANGroup(ctx, site)
}

func (c *ApiClient) GetWLANGroup(ctx context.Context, site, id string) (*WLANGroup, error) {
	return c.getWLANGroup(ctx, site, id)
}

func (c *ApiClient) DeleteWLANGroup(ctx context.Context, site, id string) error {
	return c.deleteWLANGroup(ctx, site, id)
}

func (c *ApiClient) CreateWLANGroup(
	ctx context.Context,
	site string,
	d *WLANGroup,
) (*WLANGroup, error) {
	return c.createWLANGroup(ctx, site, d)
}

func (c *ApiClient) UpdateWLANGroup(
	ctx context.Context,
	site string,
	d *WLANGroup,
) (*WLANGroup, error) {
	return c.updateWLANGroup(ctx, site, d)
}
