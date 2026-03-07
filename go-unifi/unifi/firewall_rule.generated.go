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

type FirewallRule struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	Action                string   `json:"action,omitempty"` // drop|reject|accept
	DstAddress            string   `json:"dst_address,omitempty"`
	DstAddressIPV6        string   `json:"dst_address_ipv6,omitempty"`
	DstFirewallGroupIDs   []string `json:"dst_firewallgroup_ids,omitempty"` // [\d\w]+
	DstNetworkID          string   `json:"dst_networkconf_id"`              // [\d\w]+|^$
	DstNetworkType        string   `json:"dst_networkconf_type,omitempty"`  // ADDRv4|NETv4
	DstPort               string   `json:"dst_port,omitempty"`
	Enabled               bool     `json:"enabled"`
	ICMPTypename          string   `json:"icmp_typename"`   // ^$|address-mask-reply|address-mask-request|any|communication-prohibited|destination-unreachable|echo-reply|echo-request|fragmentation-needed|host-precedence-violation|host-prohibited|host-redirect|host-unknown|host-unreachable|ip-header-bad|network-prohibited|network-redirect|network-unknown|network-unreachable|parameter-problem|port-unreachable|precedence-cutoff|protocol-unreachable|redirect|required-option-missing|router-advertisement|router-solicitation|source-quench|source-route-failed|time-exceeded|timestamp-reply|timestamp-request|TOS-host-redirect|TOS-host-unreachable|TOS-network-redirect|TOS-network-unreachable|ttl-zero-during-reassembly|ttl-zero-during-transit
	ICMPv6Typename        string   `json:"icmpv6_typename"` // ^$|address-unreachable|bad-header|beyond-scope|communication-prohibited|destination-unreachable|echo-reply|echo-request|failed-policy|neighbor-advertisement|neighbor-solicitation|no-route|packet-too-big|parameter-problem|port-unreachable|redirect|reject-route|router-advertisement|router-solicitation|time-exceeded|ttl-zero-during-reassembly|ttl-zero-during-transit|unknown-header-type|unknown-option
	IPSec                 string   `json:"ipsec"`           // match-ipsec|match-none|^$
	Logging               bool     `json:"logging"`
	Name                  string   `json:"name,omitempty"` // .{1,128}
	Protocol              string   `json:"protocol"`       // ^$|all|([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])|tcp_udp|ah|ax.25|dccp|ddp|egp|eigrp|encap|esp|etherip|fc|ggp|gre|hip|hmp|icmp|idpr-cmtp|idrp|igmp|igp|ip|ipcomp|ipencap|ipip|ipv6|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|isis|iso-tp4|l2tp|manet|mobility-header|mpls-in-ip|ospf|pim|pup|rdp|rohc|rspf|rsvp|sctp|shim6|skip|st|tcp|udp|udplite|vmtp|vrrp|wesp|xns-idp|xtp
	ProtocolMatchExcepted bool     `json:"protocol_match_excepted"`
	ProtocolV6            string   `json:"protocol_v6"`                  // ^$|([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])|ah|all|dccp|eigrp|esp|gre|icmpv6|ipcomp|ipv6|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|isis|l2tp|manet|mobility-header|mpls-in-ip|ospf|pim|rsvp|sctp|shim6|tcp|tcp_udp|udp|vrrp
	RuleIndex             *int64   `json:"rule_index,omitempty"`         // 2[0-9]{3,4}|4[0-9]{3,4}
	Ruleset               string   `json:"ruleset,omitempty"`            // WAN_IN|WAN_OUT|WAN_LOCAL|LAN_IN|LAN_OUT|LAN_LOCAL|GUEST_IN|GUEST_OUT|GUEST_LOCAL|WANv6_IN|WANv6_OUT|WANv6_LOCAL|LANv6_IN|LANv6_OUT|LANv6_LOCAL|GUESTv6_IN|GUESTv6_OUT|GUESTv6_LOCAL
	SettingPreference     string   `json:"setting_preference,omitempty"` // auto|manual
	SrcAddress            string   `json:"src_address,omitempty"`
	SrcAddressIPV6        string   `json:"src_address_ipv6,omitempty"`
	SrcFirewallGroupIDs   []string `json:"src_firewallgroup_ids,omitempty"` // [\d\w]+
	SrcMACAddress         string   `json:"src_mac_address"`                 // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$|^$
	SrcNetworkID          string   `json:"src_networkconf_id"`              // [\d\w]+|^$
	SrcNetworkType        string   `json:"src_networkconf_type,omitempty"`  // ADDRv4|NETv4
	SrcPort               string   `json:"src_port,omitempty"`
	StateEstablished      bool     `json:"state_established"`
	StateInvalid          bool     `json:"state_invalid"`
	StateNew              bool     `json:"state_new"`
	StateRelated          bool     `json:"state_related"`
}

func (dst *FirewallRule) UnmarshalJSON(b []byte) error {
	type Alias FirewallRule
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

func (c *ApiClient) listFirewallRule(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]FirewallRule, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []FirewallRule `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/firewallrule", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getFirewallRule(
	ctx context.Context,
	site string,
	id string,
) (*FirewallRule, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []FirewallRule `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/firewallrule/%s", site, id),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}
	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *ApiClient) deleteFirewallRule(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/firewallrule/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createFirewallRule(
	ctx context.Context,
	site string,
	d *FirewallRule,
) (*FirewallRule, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []FirewallRule `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/firewallrule", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}

func (c *ApiClient) updateFirewallRule(
	ctx context.Context,
	site string,
	d *FirewallRule,
) (*FirewallRule, error) {
	var respBody struct {
		Meta meta           `json:"meta"`
		Data []FirewallRule `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/firewallrule/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}
