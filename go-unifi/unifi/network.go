package unifi

import (
	"context"
	"fmt"
	"slices"
)

func (c *ApiClient) DeleteNetwork(ctx context.Context, site, id, name string) error {
	err := c.do(ctx, "DELETE", fmt.Sprintf("api/s/%s/rest/networkconf/%s", site, id), struct {
		Name string `json:"name"`
	}{
		Name: name,
	}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) ListNetwork(ctx context.Context, site string, params ...[]struct {
	key string
	val string
},
) ([]Network, error) {
	return c.listNetwork(ctx, site)
}

func (c *ApiClient) GetNetwork(ctx context.Context, site, id string) (*Network, error) {
	return c.getNetwork(ctx, site, id)
}

func (c *ApiClient) GetNetworkByName(ctx context.Context, site, name string) (*Network, error) {
	networks, err := c.listNetwork(ctx, site)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(networks, func(n Network) bool {
		if n.Name == nil {
			return false
		}
		return *n.Name == name
	})
	if i < 0 {
		return nil, fmt.Errorf("network with name %s not found", name)
	}
	network := networks[i]
	return &network, nil
}

func (c *ApiClient) CreateNetwork(ctx context.Context, site string, d *Network) (*Network, error) {
	return c.createNetwork(ctx, site, d)
}

func (c *ApiClient) UpdateNetwork(ctx context.Context, site string, d *Network) (*Network, error) {
	return c.updateNetwork(ctx, site, d)
}
