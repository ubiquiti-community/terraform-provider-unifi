// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
	_ strings.Builder
)

type FirewallPolicy struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Action                string                     `json:"action,omitempty"`                // ALLOW|BLOCK|REJECT
	ConnectionStateType   string                     `json:"connection_state_type,omitempty"` // ALL|RESPOND_ONLY
	ConnectionStates      []string                   `json:"connection_states"`
	CreateAllowRespond    bool                       `json:"create_allow_respond"`
	Description           string                     `json:"description,omitempty"`
	Destination           *FirewallPolicyDestination `json:"destination,omitempty"`
	Enabled               bool                       `json:"enabled"`
	ICMPTypename          string                     `json:"icmp_typename,omitempty"`    // ANY|SPECIFIC|LIST|OBJECT
	ICMPV6Typename        string                     `json:"icmp_v6_typename,omitempty"` // ANY|SPECIFIC|LIST|OBJECT
	IPVersion             string                     `json:"ip_version,omitempty"`       // BOTH|IPV4|IPV6
	Index                 *int64                     `json:"index,omitempty"`            // [1-9][0-9]+
	Logging               bool                       `json:"logging"`
	MatchIPSec            bool                       `json:"match_ip_sec"`
	MatchOppositeProtocol bool                       `json:"match_opposite_protocol"`
	Name                  string                     `json:"name,omitempty"`
	Predefined            bool                       `json:"predefined"`
	Protocol              string                     `json:"protocol,omitempty"` // all|tcp|udp|tcp_udp
	Schedule              *FirewallPolicySchedule    `json:"schedule,omitempty"`
	Source                *FirewallPolicySource      `json:"source,omitempty"`
}

func (dst *FirewallPolicy) UnmarshalJSON(b []byte) error {
	type Alias FirewallPolicy
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type FirewallPolicyDestination struct {
	IPs                []string `json:"ips,omitempty"`
	MatchMAC           bool     `json:"match_mac"`
	MatchOppositeIPs   bool     `json:"match_opposite_ips"`
	MatchOppositePorts bool     `json:"match_opposite_ports"`
	MatchingTarget     string   `json:"matching_target,omitempty"`      // ANY|DEVICE|IP|NETWORK|MAC
	MatchingTargetType string   `json:"matching_target_type,omitempty"` // ANY|SPECIFIC|LIST|OBJECT
	Port               *int64   `json:"port,omitempty"`                 // [1-9][0-9]{0,4}
	PortGroupID        string   `json:"port_group_id,omitempty"`
	PortMatchingType   string   `json:"port_matching_type,omitempty"` // ANY|SPECIFIC|LIST|OBJECT
	ZoneID             string   `json:"zone_id,omitempty"`
}

func (dst *FirewallPolicyDestination) UnmarshalJSON(b []byte) error {
	type Alias FirewallPolicyDestination
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type FirewallPolicySchedule struct {
	Date           string   `json:"date,omitempty"`
	Mode           string   `json:"mode,omitempty"`           // ALWAYS|EVERY_DAY|EVERY_WEEK|ONE_TIME_ONLY
	RepeatOnDays   []string `json:"repeat_on_days,omitempty"` // mon|tue|wed|thu|fri|sat|sun
	TimeAllDay     bool     `json:"time_all_day"`
	TimeRangeEnd   string   `json:"time_range_end,omitempty"`
	TimeRangeStart string   `json:"time_range_start,omitempty"`
}

func (dst *FirewallPolicySchedule) UnmarshalJSON(b []byte) error {
	type Alias FirewallPolicySchedule
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type FirewallPolicySource struct {
	IPs                []string `json:"ips,omitempty"`
	MatchMAC           bool     `json:"match_mac"`
	MatchOppositeIPs   bool     `json:"match_opposite_ips"`
	MatchOppositePorts bool     `json:"match_opposite_ports"`
	MatchingTarget     string   `json:"matching_target,omitempty"`      // ANY|DEVICE|IP|NETWORK|MAC
	MatchingTargetType string   `json:"matching_target_type,omitempty"` // ANY|SPECIFIC|LIST|OBJECT
	Port               *int64   `json:"port,omitempty"`                 // [1-9][0-9]{0,4}
	PortGroupID        string   `json:"port_group_id,omitempty"`
	PortMatchingType   string   `json:"port_matching_type,omitempty"` // ANY|SPECIFIC|LIST|OBJECT
	ZoneID             string   `json:"zone_id,omitempty"`
}

func (dst *FirewallPolicySource) UnmarshalJSON(b []byte) error {
	type Alias FirewallPolicySource
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

func (c *ApiClient) listFirewallPolicy(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]FirewallPolicy, error) {
	var respBody []FirewallPolicy

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/firewall-policies", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func (c *ApiClient) getFirewallPolicy(
	ctx context.Context,
	site string,
	id string,
) (*FirewallPolicy, error) {
	respBody, err := c.listFirewallPolicy(ctx, site)
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

func (c *ApiClient) deleteFirewallPolicy(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("v2/api/site/%s/firewall-policies/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createFirewallPolicy(
	ctx context.Context,
	site string,
	d *FirewallPolicy,
) (*FirewallPolicy, error) {
	var respBody FirewallPolicy

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("v2/api/site/%s/firewall-policies", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

func (c *ApiClient) updateFirewallPolicy(
	ctx context.Context,
	site string,
	d *FirewallPolicy,
) (*FirewallPolicy, error) {
	var respBody FirewallPolicy
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("v2/api/site/%s/firewall-policies/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}
