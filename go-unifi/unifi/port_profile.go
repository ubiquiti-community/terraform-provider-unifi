package unifi

import (
	"context"
)

func (c *ApiClient) ListPortProfile(ctx context.Context, site string) ([]PortProfile, error) {
	return c.listPortProfile(ctx, site)
}

func (c *ApiClient) GetPortProfile(ctx context.Context, site, id string) (*PortProfile, error) {
	return c.getPortProfile(ctx, site, id)
}

func (c *ApiClient) DeletePortProfile(ctx context.Context, site, id string) error {
	return c.deletePortProfile(ctx, site, id)
}

func (c *ApiClient) CreatePortProfile(
	ctx context.Context,
	site string,
	d *PortProfile,
) (*PortProfile, error) {
	return c.createPortProfile(ctx, site, d)
}

func (c *ApiClient) UpdatePortProfile(
	ctx context.Context,
	site string,
	d *PortProfile,
) (*PortProfile, error) {
	return c.updatePortProfile(ctx, site, d)
}
