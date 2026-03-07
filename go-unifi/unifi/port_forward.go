package unifi

import "context"

func (c *ApiClient) ListPortForward(ctx context.Context, site string) ([]PortForward, error) {
	return c.listPortForward(ctx, site)
}

func (c *ApiClient) GetPortForward(ctx context.Context, site, id string) (*PortForward, error) {
	return c.getPortForward(ctx, site, id)
}

func (c *ApiClient) DeletePortForward(ctx context.Context, site, id string) error {
	return c.deletePortForward(ctx, site, id)
}

func (c *ApiClient) CreatePortForward(
	ctx context.Context,
	site string,
	d *PortForward,
) (*PortForward, error) {
	return c.createPortForward(ctx, site, d)
}

func (c *ApiClient) UpdatePortForward(
	ctx context.Context,
	site string,
	d *PortForward,
) (*PortForward, error) {
	return c.updatePortForward(ctx, site, d)
}
