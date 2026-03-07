// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
)

func (c *ApiClient) ListFirewallPolicy(ctx context.Context, site string) ([]FirewallPolicy, error) {
	return c.listFirewallPolicy(ctx, site)
}

func (c *ApiClient) GetFirewallPolicy(ctx context.Context, site, id string) (*FirewallPolicy, error) {
	return c.getFirewallPolicy(ctx, site, id)
}

func (c *ApiClient) DeleteFirewallPolicy(ctx context.Context, site, id string) error {
	return c.deleteFirewallPolicy(ctx, site, id)
}

func (c *ApiClient) CreateFirewallPolicy(ctx context.Context, site string, d *FirewallPolicy) (*FirewallPolicy, error) {
	return c.createFirewallPolicy(ctx, site, d)
}

func (c *ApiClient) UpdateFirewallPolicy(ctx context.Context, site string, d *FirewallPolicy) (*FirewallPolicy, error) {
	return c.updateFirewallPolicy(ctx, site, d)
}
