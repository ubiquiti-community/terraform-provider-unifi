// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListOSPFRouter(ctx context.Context, site string) ([]OSPFRouter, error) {
	return c.listOSPFRouter(ctx, site)
}

func (c *ApiClient) GetOSPFRouter(ctx context.Context, site, id string) (*OSPFRouter, error) {
	respBody, err := c.listOSPFRouter(ctx, site)
	if err != nil {
		return nil, err
	}

	if len(respBody) == 0 {
		return nil, &NotFoundError{}
	}

	for _, val := range respBody {
		if val.ID == id {
			return &val, nil
		}
	}

	return nil, &NotFoundError{}
}

func (c *ApiClient) DeleteOSPFRouter(ctx context.Context, site, id string) error {
	return c.deleteOSPFRouter(ctx, site, id)
}

func (c *ApiClient) CreateOSPFRouter(ctx context.Context, site string, d *OSPFRouter) (*OSPFRouter, error) {
	return c.createOSPFRouter(ctx, site, d)
}

func (c *ApiClient) UpdateOSPFRouter(ctx context.Context, site string, d *OSPFRouter) (*OSPFRouter, error) {
	return c.updateOSPFRouter(ctx, site, d)
}
