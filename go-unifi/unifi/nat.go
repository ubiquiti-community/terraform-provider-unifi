// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListNat(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]Nat, error) {
	return c.listNat(ctx, site, query...)
}

func (c *ApiClient) GetNat(
	ctx context.Context,
	site,
	id string,
) (*Nat, error) {
	return c.getNat(ctx, site, id)
}

func (c *ApiClient) DeleteNat(ctx context.Context, site, id string) error {
	return c.deleteNat(ctx, site, id)
}

func (c *ApiClient) CreateNat(ctx context.Context, site string, d *Nat) (*Nat, error) {
	return c.createNat(ctx, site, d)
}

func (c *ApiClient) UpdateNat(ctx context.Context, site string, d *Nat) (*Nat, error) {
	return c.updateNat(ctx, site, d)
}
