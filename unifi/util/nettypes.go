package util

import (
	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
)

// This file provides read-side conversion helpers that mirror StringValueOrNull
// for the custom string types from terraform-plugin-framework-nettypes. The
// UniFi API represents missing values as empty strings, so every helper maps an
// empty string to the corresponding null value.
//
// Write-side conversion does not need dedicated helpers: every nettypes value
// embeds basetypes.StringValue, so .ValueString(), .IsNull() and .IsUnknown()
// are available directly on the model fields.

// MACValueOrNull returns a hwtypes.MACAddress, null when the string is empty.
func MACValueOrNull(val string) hwtypes.MACAddress {
	if val == "" {
		return hwtypes.NewMACAddressNull()
	}
	return hwtypes.NewMACAddressValue(val)
}

// IPv4ValueOrNull returns an iptypes.IPv4Address, null when the string is empty.
func IPv4ValueOrNull(val string) iptypes.IPv4Address {
	if val == "" {
		return iptypes.NewIPv4AddressNull()
	}
	return iptypes.NewIPv4AddressValue(val)
}

// IPv6ValueOrNull returns an iptypes.IPv6Address, null when the string is empty.
func IPv6ValueOrNull(val string) iptypes.IPv6Address {
	if val == "" {
		return iptypes.NewIPv6AddressNull()
	}
	return iptypes.NewIPv6AddressValue(val)
}

// IPValueOrNull returns an iptypes.IPAddress (IPv4 or IPv6), null when empty.
func IPValueOrNull(val string) iptypes.IPAddress {
	if val == "" {
		return iptypes.NewIPAddressNull()
	}
	return iptypes.NewIPAddressValue(val)
}

// IPv4PrefixOrNull returns a cidrtypes.IPv4Prefix, null when the string is empty.
func IPv4PrefixOrNull(val string) cidrtypes.IPv4Prefix {
	if val == "" {
		return cidrtypes.NewIPv4PrefixNull()
	}
	return cidrtypes.NewIPv4PrefixValue(val)
}

// IPv6PrefixOrNull returns a cidrtypes.IPv6Prefix, null when the string is empty.
func IPv6PrefixOrNull(val string) cidrtypes.IPv6Prefix {
	if val == "" {
		return cidrtypes.NewIPv6PrefixNull()
	}
	return cidrtypes.NewIPv6PrefixValue(val)
}

// Pointer variants for APIs that expose *string fields. A nil or empty-string
// pointer maps to the corresponding null value.

// IPv4PtrValueOrNull returns an iptypes.IPv4Address from a *string.
func IPv4PtrValueOrNull(val *string) iptypes.IPv4Address {
	if val == nil {
		return iptypes.NewIPv4AddressNull()
	}
	return IPv4ValueOrNull(*val)
}

// IPv6PtrValueOrNull returns an iptypes.IPv6Address from a *string.
func IPv6PtrValueOrNull(val *string) iptypes.IPv6Address {
	if val == nil {
		return iptypes.NewIPv6AddressNull()
	}
	return IPv6ValueOrNull(*val)
}

// IPPtrValueOrNull returns an iptypes.IPAddress from a *string.
func IPPtrValueOrNull(val *string) iptypes.IPAddress {
	if val == nil {
		return iptypes.NewIPAddressNull()
	}
	return IPValueOrNull(*val)
}
