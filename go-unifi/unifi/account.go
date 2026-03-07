// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListAccount(ctx context.Context, site string) ([]Account, error) {
	return c.listAccount(ctx, site)
}

func (c *ApiClient) GetAccount(ctx context.Context, site, id string) (*Account, error) {
	return c.getAccount(ctx, site, id)
}

func (c *ApiClient) DeleteAccount(ctx context.Context, site, id string) error {
	return c.deleteAccount(ctx, site, id)
}

func (c *ApiClient) CreateAccount(ctx context.Context, site string, d *Account) (*Account, error) {
	return c.createAccount(ctx, site, d)
}

func (c *ApiClient) UpdateAccount(ctx context.Context, site string, d *Account) (*Account, error) {
	return c.updateAccount(ctx, site, d)
}
