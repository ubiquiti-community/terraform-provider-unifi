// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
)

type Ips struct {
	BaseSetting

	AdvancedFilteringPreference         string                 `json:"advanced_filtering_preference,omitempty"` // |manual|disabled
	ContentFilteringBlockingPageEnabled bool                   `json:"content_filtering_blocking_page_enabled"`
	EnabledCategories                   []string               `json:"enabled_categories,omitempty"` // emerging-activex|emerging-attackresponse|botcc|emerging-chat|ciarmy|compromised|emerging-dns|emerging-dos|dshield|emerging-exploit|emerging-ftp|emerging-games|emerging-icmp|emerging-icmpinfo|emerging-imap|emerging-inappropriate|emerging-info|emerging-malware|emerging-misc|emerging-mobile|emerging-netbios|emerging-p2p|emerging-policy|emerging-pop3|emerging-rpc|emerging-scada|emerging-scan|emerging-shellcode|emerging-smtp|emerging-snmp|emerging-sql|emerging-telnet|emerging-tftp|tor|emerging-useragent|emerging-voip|emerging-webapps|emerging-webclient|emerging-webserver|emerging-worm|exploit-kit|adware-pup|botcc-portgrouped|phishing|threatview-cs-c2|3coresec|chat|coinminer|current-events|drop|hunting|icmp-info|inappropriate|info|ja3|policy|scada|dark-web-blocker-list|malicious-hosts
	EnabledNetworks                     []string               `json:"enabled_networks,omitempty"`
	Honeypot                            []SettingIpsHoneypot   `json:"honeypot,omitempty"`
	HoneypotEnabled                     bool                   `json:"honeypot_enabled"`
	IPsMode                             string                 `json:"ips_mode,omitempty"` // ids|ips|ipsInline|disabled
	MemoryOptimized                     bool                   `json:"memory_optimized"`
	RestrictTorrents                    bool                   `json:"restrict_torrents"`
	Suppression                         *SettingIpsSuppression `json:"suppression,omitempty"`
}

func (dst *Ips) UnmarshalJSON(b []byte) error {
	type Alias Ips
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	// First unmarshal base setting
	if err := json.Unmarshal(b, &dst.BaseSetting); err != nil {
		return fmt.Errorf("unable to unmarshal base setting: %w", err)
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type SettingIpsAlerts struct {
	Category  string               `json:"category,omitempty"`
	Gid       *int64               `json:"gid,omitempty"`
	ID        *int64               `json:"id,omitempty"`
	Signature string               `json:"signature,omitempty"`
	Tracking  []SettingIpsTracking `json:"tracking,omitempty"`
	Type      string               `json:"type,omitempty"` // all|track
}

func (dst *SettingIpsAlerts) UnmarshalJSON(b []byte) error {
	type Alias SettingIpsAlerts
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

type SettingIpsHoneypot struct {
	IPAddress string `json:"ip_address,omitempty"`
	NetworkID string `json:"network_id,omitempty"`
	Version   string `json:"version,omitempty"` // v4|v6
}

func (dst *SettingIpsHoneypot) UnmarshalJSON(b []byte) error {
	type Alias SettingIpsHoneypot
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

type SettingIpsSuppression struct {
	Alerts    []SettingIpsAlerts    `json:"alerts,omitempty"`
	Whitelist []SettingIpsWhitelist `json:"whitelist,omitempty"`
}

func (dst *SettingIpsSuppression) UnmarshalJSON(b []byte) error {
	type Alias SettingIpsSuppression
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

type SettingIpsTracking struct {
	Direction string `json:"direction,omitempty"` // both|src|dest
	Mode      string `json:"mode,omitempty"`      // ip|subnet|network
	Value     string `json:"value,omitempty"`
}

func (dst *SettingIpsTracking) UnmarshalJSON(b []byte) error {
	type Alias SettingIpsTracking
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

type SettingIpsWhitelist struct {
	Direction string `json:"direction,omitempty"` // both|src|dest
	Mode      string `json:"mode,omitempty"`      // ip|subnet|network
	Value     string `json:"value,omitempty"`
}

func (dst *SettingIpsWhitelist) UnmarshalJSON(b []byte) error {
	type Alias SettingIpsWhitelist
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
