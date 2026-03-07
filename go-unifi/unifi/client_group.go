// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListClientGroup(ctx context.Context, site string) ([]ClientGroup, error) {
	return c.listClientGroup(ctx, site)
}

func (c *ApiClient) GetClientGroup(ctx context.Context, site, id string) (*ClientGroup, error) {
	return c.getClientGroup(ctx, site, id)
}

func (c *ApiClient) DeleteClientGroup(ctx context.Context, site, id string) error {
	return c.deleteClientGroup(ctx, site, id)
}

func (c *ApiClient) CreateClientGroup(ctx context.Context, site string, d *ClientGroup) (*ClientGroup, error) {
	return c.createClientGroup(ctx, site, d)
}

func (c *ApiClient) UpdateClientGroup(ctx context.Context, site string, d *ClientGroup) (*ClientGroup, error) {
	return c.updateClientGroup(ctx, site, d)
}
