// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListTrafficRoute(ctx context.Context, site string) ([]TrafficRoute, error) {
	return c.listTrafficRoute(ctx, site)
}

func (c *ApiClient) GetTrafficRoute(ctx context.Context, site, id string) (*TrafficRoute, error) {
	respBody, err := c.listTrafficRoute(ctx, site)
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

func (c *ApiClient) DeleteTrafficRoute(ctx context.Context, site, id string) error {
	return c.deleteTrafficRoute(ctx, site, id)
}

func (c *ApiClient) CreateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.createTrafficRoute(ctx, site, d)
}

func (c *ApiClient) UpdateTrafficRoute(ctx context.Context, site string, d *TrafficRoute) (*TrafficRoute, error) {
	return c.updateTrafficRoute(ctx, site, d)
}
