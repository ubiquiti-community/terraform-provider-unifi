// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListWLAN(ctx context.Context, site string) ([]WLAN, error) {
	return c.listWLAN(ctx, site)
}

func (c *ApiClient) GetWLAN(ctx context.Context, site, id string) (*WLAN, error) {
	return c.getWLAN(ctx, site, id)
}

func (c *ApiClient) GetWLANByName(ctx context.Context, site, name string) (*WLAN, error) {
	wlans, err := c.ListWLAN(ctx, site)
	if err != nil {
		return nil, err
	}

	for _, w := range wlans {
		if w.Name == name {
			return &w, nil
		}
	}

	return nil, &NotFoundError{
		Type:  "WLAN",
		Attr:  "Name",
		Value: name,
	}
}

func (c *ApiClient) DeleteWLAN(ctx context.Context, site, id string) error {
	return c.deleteWLAN(ctx, site, id)
}

func (c *ApiClient) CreateWLAN(ctx context.Context, site string, d *WLAN) (*WLAN, error) {
	return c.createWLAN(ctx, site, d)
}

func (c *ApiClient) UpdateWLAN(ctx context.Context, site string, d *WLAN) (*WLAN, error) {
	return c.updateWLAN(ctx, site, d)
}
