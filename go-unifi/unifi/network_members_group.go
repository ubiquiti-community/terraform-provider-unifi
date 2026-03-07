package unifi

import (
	"context"
	"fmt"
)

// This is a v2 API object, manually coded.

type NetworkMembersGroup struct {
	ID      string   `json:"id,omitempty"`
	Name    string   `json:"name"`
	Members []string `json:"members"`
	Type    string   `json:"type"`
}

func (c *ApiClient) ListNetworkMembersGroups(ctx context.Context, site string) ([]NetworkMembersGroup, error) {
	var respBody []NetworkMembersGroup

	err := c.do(ctx, "GET", fmt.Sprintf("v2/api/site/%s/network-members-groups", site), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *ApiClient) GetNetworkMembersGroup(ctx context.Context, site string, id string) (*NetworkMembersGroup, error) {
	var respBody NetworkMembersGroup

	err := c.do(ctx, "GET", fmt.Sprintf("v2/api/site/%s/network-members-group/%s", site, id), nil, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) CreateNetworkMembersGroup(ctx context.Context, site string, d *NetworkMembersGroup) (*NetworkMembersGroup, error) {
	var respBody NetworkMembersGroup

	err := c.do(ctx, "POST", fmt.Sprintf("v2/api/site/%s/network-members-groups", site), d, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) UpdateNetworkMembersGroup(ctx context.Context, site string, d *NetworkMembersGroup) (*NetworkMembersGroup, error) {
	var respBody NetworkMembersGroup

	err := c.do(ctx, "PUT", fmt.Sprintf("v2/api/site/%s/network-members-group/%s", site, d.ID), d, &respBody)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) DeleteNetworkMembersGroup(ctx context.Context, site string, id string) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("v2/api/site/%s/network-members-group/%s", site, id), nil, nil)
}
