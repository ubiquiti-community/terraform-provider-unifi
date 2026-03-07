// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListBroadcastGroup(ctx context.Context, site string) ([]BroadcastGroup, error) {
	return c.listBroadcastGroup(ctx, site)
}

func (c *ApiClient) GetBroadcastGroup(ctx context.Context, site, id string) (*BroadcastGroup, error) {
	return c.getBroadcastGroup(ctx, site, id)
}

func (c *ApiClient) DeleteBroadcastGroup(ctx context.Context, site, id string) error {
	return c.deleteBroadcastGroup(ctx, site, id)
}

func (c *ApiClient) CreateBroadcastGroup(
	ctx context.Context,
	site string,
	d *BroadcastGroup,
) (*BroadcastGroup, error) {
	return c.createBroadcastGroup(ctx, site, d)
}

func (c *ApiClient) UpdateBroadcastGroup(
	ctx context.Context,
	site string,
	d *BroadcastGroup,
) (*BroadcastGroup, error) {
	return c.updateBroadcastGroup(ctx, site, d)
}
