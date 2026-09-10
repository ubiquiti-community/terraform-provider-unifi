package unifi

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// testNetworkIPv6Object builds a known ipv6 object with the given
// interface_type and every other leaf at the null/zero value the flat
// attributes carried in a minimal model.
func testNetworkIPv6Object(interfaceType types.String) types.Object {
	return types.ObjectValueMust(networkIPv6AttrTypes(), map[string]attr.Value{
		"interface_type":            interfaceType,
		"client_address_assignment": types.StringNull(),
		"static_subnet":             types.StringNull(),
		"aliases":                   types.ListNull(types.StringType),
		"ra": types.ObjectValueMust(networkIPv6RAAttrTypes(), map[string]attr.Value{
			"enabled":            types.BoolValue(false),
			"priority":           types.StringNull(),
			"preferred_lifetime": timetypes.NewGoDurationNull(),
			"valid_lifetime":     timetypes.NewGoDurationNull(),
		}),
		"pd": types.ObjectValueMust(networkIPv6PDAttrTypes(), map[string]attr.Value{
			"interface":             types.StringNull(),
			"prefixid":              types.StringNull(),
			"start":                 types.StringNull(),
			"stop":                  types.StringNull(),
			"auto_prefixid_enabled": types.BoolValue(false),
		}),
	})
}

// durationOf parses a GoDuration attribute value; configured values keep the
// practitioner's spelling ("4h") while controller-derived ones are normalized
// ("4h0m0s"), so comparisons go through time.Duration.
func durationOf(t *testing.T, v attr.Value) time.Duration {
	t.Helper()
	d, err := time.ParseDuration(attrAs[timetypes.GoDuration](t, v).ValueString())
	if err != nil {
		t.Fatalf("parse duration %v: %v", v, err)
	}
	return d
}

// TestNetworkSchemas_validateImplementation guards the object-level defaults
// (which carry unknown leaves) and the nested attribute layout against the
// framework's own implementation checks for both the resource and the data
// source.
func TestNetworkSchemas_validateImplementation(t *testing.T) {
	ctx := context.Background()

	var rs fwresource.SchemaResponse
	(&networkResource{}).Schema(ctx, fwresource.SchemaRequest{}, &rs)
	if d := rs.Schema.ValidateImplementation(ctx); d.HasError() {
		t.Errorf("resource schema: %v", d)
	}
	if rs.Schema.Version != 2 {
		t.Errorf("resource schema Version = %d, want 2", rs.Schema.Version)
	}

	var ds fwdatasource.SchemaResponse
	(&networkDataSource{}).Schema(ctx, fwdatasource.SchemaRequest{}, &ds)
	if d := ds.Schema.ValidateImplementation(ctx); d.HasError() {
		t.Errorf("data source schema: %v", d)
	}
	for _, name := range []string{"ipv6", "wan", "dhcp_server", "dhcp_v6_server"} {
		if _, ok := ds.Schema.Attributes[name]; !ok {
			t.Errorf("data source schema missing %q", name)
		}
	}
}

// TestNetworkUpgradeState_nestsPrefixedGroups guards the v1 -> v2 schema
// upgrade: the flat ipv6_* attributes move under ipv6 (with ra and pd
// sub-objects), dhcp_server.dns_*/ntp_* under dhcp_server.dns/ntp and
// dhcp_v6_server.dns_* under dhcp_v6_server.dns. The v0 upgrader must apply
// the same rewrite on top of its duration conversion.
func TestNetworkUpgradeState_nestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	ups := r.UpgradeState(ctx)
	for _, v := range []int64{0, 1} {
		if _, ok := ups[v]; !ok {
			t.Fatalf("no upgrader registered for schema version %d", v)
		}
	}

	upgrade := func(t *testing.T, version int64, prior string) map[string]tftypes.Value {
		t.Helper()
		resp := &fwresource.UpgradeStateResponse{}
		ups[version].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: []byte(prior)},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("upgrade from v%d failed: %v", version, resp.Diagnostics)
		}
		val, err := resp.DynamicValue.Unmarshal(schemaType)
		if err != nil {
			t.Fatalf("unmarshal upgraded value: %v", err)
		}
		var root map[string]tftypes.Value
		if err := val.As(&root); err != nil {
			t.Fatalf("as object: %v", err)
		}
		return root
	}
	obj := func(t *testing.T, v tftypes.Value, name string) map[string]tftypes.Value {
		t.Helper()
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v (value %v)", name, err, v)
		}
		return m
	}
	str := func(t *testing.T, v tftypes.Value, name, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil || s != want {
			t.Errorf("%s = %v (%v), want %q", name, v, err, want)
		}
	}
	boolean := func(t *testing.T, v tftypes.Value, name string, want bool) {
		t.Helper()
		var b bool
		if err := v.As(&b); err != nil || b != want {
			t.Errorf("%s = %v (%v), want %v", name, v, err, want)
		}
	}
	listLen := func(t *testing.T, v tftypes.Value, name string, want int) {
		t.Helper()
		var l []tftypes.Value
		if err := v.As(&l); err != nil || len(l) != want {
			t.Errorf("%s = %v (%v), want %d entries", name, v, err, want)
		}
	}
	assertNested := func(t *testing.T, root map[string]tftypes.Value) {
		t.Helper()
		for _, flat := range []string{
			"ipv6_interface_type", "ipv6_ra", "ipv6_ra_priority", "ipv6_pd_start", "ipv6_aliases",
		} {
			if _, exists := root[flat]; exists {
				t.Errorf("flat attribute %q survived the upgrade", flat)
			}
		}
		str(t, root["name"], "name", "iot")

		v6 := obj(t, root["ipv6"], "ipv6")
		str(t, v6["interface_type"], "ipv6.interface_type", "static")
		str(t, v6["client_address_assignment"], "ipv6.client_address_assignment", "slaac")
		str(t, v6["static_subnet"], "ipv6.static_subnet", "fd00::1/64")
		if !v6["aliases"].IsNull() {
			t.Errorf("ipv6.aliases = %v, want null", v6["aliases"])
		}
		ra := obj(t, v6["ra"], "ipv6.ra")
		boolean(t, ra["enabled"], "ipv6.ra.enabled", true)
		str(t, ra["priority"], "ipv6.ra.priority", "high")
		str(t, ra["preferred_lifetime"], "ipv6.ra.preferred_lifetime", "4h0m0s")
		str(t, ra["valid_lifetime"], "ipv6.ra.valid_lifetime", "24h0m0s")
		pd := obj(t, v6["pd"], "ipv6.pd")
		str(t, pd["interface"], "ipv6.pd.interface", "wan")
		str(t, pd["prefixid"], "ipv6.pd.prefixid", "1")
		str(t, pd["start"], "ipv6.pd.start", "::2")
		str(t, pd["stop"], "ipv6.pd.stop", "::7d1")
		boolean(t, pd["auto_prefixid_enabled"], "ipv6.pd.auto_prefixid_enabled", false)

		dhcp := obj(t, root["dhcp_server"], "dhcp_server")
		for _, flat := range []string{"dns_enabled", "dns_servers", "ntp_enabled", "ntp_servers"} {
			if _, exists := dhcp[flat]; exists {
				t.Errorf("flat attribute dhcp_server.%q survived the upgrade", flat)
			}
		}
		str(t, dhcp["start"], "dhcp_server.start", "10.0.2.10")
		str(t, dhcp["leasetime"], "dhcp_server.leasetime", "24h0m0s")
		dns := obj(t, dhcp["dns"], "dhcp_server.dns")
		boolean(t, dns["enabled"], "dhcp_server.dns.enabled", true)
		listLen(t, dns["servers"], "dhcp_server.dns.servers", 1)
		ntp := obj(t, dhcp["ntp"], "dhcp_server.ntp")
		boolean(t, ntp["enabled"], "dhcp_server.ntp.enabled", false)
		if !ntp["servers"].IsNull() {
			t.Errorf("dhcp_server.ntp.servers = %v, want null", ntp["servers"])
		}

		v6dhcp := obj(t, root["dhcp_v6_server"], "dhcp_v6_server")
		if _, exists := v6dhcp["dns_auto"]; exists {
			t.Error("flat attribute dhcp_v6_server.dns_auto survived the upgrade")
		}
		v6dns := obj(t, v6dhcp["dns"], "dhcp_v6_server.dns")
		boolean(t, v6dns["auto"], "dhcp_v6_server.dns.auto", false)
		listLen(t, v6dns["servers"], "dhcp_v6_server.dns.servers", 1)
		str(t, v6dhcp["stop"], "dhcp_v6_server.stop", "::7d1")
	}

	t.Run("v1", func(t *testing.T) {
		root := upgrade(t, 1, `{
			"id": "net-1", "site": "default", "name": "iot", "subnet": "10.0.2.1/24", "vlan": 20,
			"ipv6_interface_type": "static", "ipv6_client_address_assignment": "slaac",
			"ipv6_static_subnet": "fd00::1/64", "ipv6_aliases": null,
			"ipv6_ra": true, "ipv6_ra_priority": "high",
			"ipv6_ra_preferred_lifetime": "4h0m0s", "ipv6_ra_valid_lifetime": "24h0m0s",
			"ipv6_pd_interface": "wan", "ipv6_pd_prefixid": "1",
			"ipv6_pd_start": "::2", "ipv6_pd_stop": "::7d1", "ipv6_pd_auto_prefixid_enabled": false,
			"dhcp_server": {
				"enabled": true, "start": "10.0.2.10", "stop": "10.0.2.200", "leasetime": "24h0m0s",
				"dns_enabled": true, "dns_servers": ["10.0.2.53"],
				"ntp_enabled": false, "ntp_servers": null,
				"wins": {"enabled": false, "addresses": null}
			},
			"dhcp_v6_server": {
				"enabled": true, "dns_auto": false, "dns_servers": ["2001:4860:4860::8888"],
				"lease": 86400, "start": "::2", "stop": "::7d1"
			}
		}`)
		assertNested(t, root)
	})

	t.Run("v0", func(t *testing.T) {
		root := upgrade(t, 0, `{
			"id": "net-1", "site": "default", "name": "iot", "subnet": "10.0.2.1/24", "vlan": 20,
			"ipv6_interface_type": "static", "ipv6_client_address_assignment": "slaac",
			"ipv6_static_subnet": "fd00::1/64",
			"ipv6_ra": true, "ipv6_ra_priority": "high",
			"ipv6_ra_preferred_lifetime": 14400, "ipv6_ra_valid_lifetime": 86400,
			"ipv6_pd_interface": "wan", "ipv6_pd_prefixid": "1",
			"ipv6_pd_start": "::2", "ipv6_pd_stop": "::7d1", "ipv6_pd_auto_prefixid_enabled": false,
			"dhcp_server": {
				"enabled": true, "start": "10.0.2.10", "stop": "10.0.2.200", "leasetime": 86400,
				"dns_enabled": true, "dns_servers": ["10.0.2.53"],
				"ntp_enabled": false, "ntp_servers": null,
				"wins": {"enabled": false, "addresses": null}
			},
			"dhcp_v6_server": {
				"enabled": true, "dns_auto": false, "dns_servers": ["2001:4860:4860::8888"],
				"lease": 86400, "start": "::2", "stop": "::7d1"
			}
		}`)
		assertNested(t, root)
	})
}

// TestNetworkNestedGroups_roundTrip checks that the nested ipv6, dhcp_server
// dns/ntp and dhcp_v6_server dns groups are written to the API struct exactly
// as the flat attributes were and rebuilt from the API response, and that a
// null group contributes the same request fields the unset flat attributes
// did.
func TestNetworkNestedGroups_roundTrip(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	strList := func(vals ...string) types.List {
		elems := make([]attr.Value, 0, len(vals))
		for _, v := range vals {
			elems = append(elems, types.StringValue(v))
		}
		return types.ListValueMust(types.StringType, elems)
	}
	option := func(enabled bool, servers types.List) types.Object {
		return types.ObjectValueMust(
			dhcpServerOptionModel{}.AttributeTypes(),
			map[string]attr.Value{
				"enabled": types.BoolValue(enabled),
				"servers": servers,
			},
		)
	}
	dhcpServer := func(dns, ntp types.Object) types.Object {
		return types.ObjectValueMust(dhcpServerModel{}.AttributeTypes(), map[string]attr.Value{
			"boot":                types.ObjectNull(dhcpBootModel{}.AttributeTypes()),
			"enabled":             types.BoolValue(true),
			"start":               types.StringValue("10.0.1.10"),
			"stop":                types.StringValue("10.0.1.200"),
			"gateway_enabled":     types.BoolValue(false),
			"conflict_checking":   types.BoolValue(true),
			"ntp":                 ntp,
			"time_offset_enabled": types.BoolValue(false),
			"dns":                 dns,
			"leasetime":           timetypes.NewGoDurationValueFromStringMust("24h"),
			"wins":                types.ObjectNull(winsModel{}.AttributeTypes()),
			"wpad_url":            types.StringNull(),
			"tftp_server":         types.StringNull(),
			"unifi_controller":    types.StringNull(),
		})
	}
	dhcpV6 := func(dns types.Object) types.Object {
		return types.ObjectValueMust(dhcpV6ServerModel{}.AttributeTypes(), map[string]attr.Value{
			"enabled": types.BoolValue(true),
			"dns":     dns,
			"lease":   types.Int64Value(86400),
			"start":   types.StringValue("::2"),
			"stop":    types.StringValue("::7d1"),
		})
	}
	base := func() *networkResourceModel {
		return &networkResourceModel{
			Name:   types.StringValue("dual"),
			Subnet: cidrtypes.NewIPv4PrefixValue("10.0.1.1/24"),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:    types.ListNull(types.StringType),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
		}
	}

	model := base()
	model.IPv6 = types.ObjectValueMust(networkIPv6AttrTypes(), map[string]attr.Value{
		"interface_type":            types.StringValue("pd"),
		"client_address_assignment": types.StringValue("slaac"),
		"static_subnet":             types.StringNull(),
		"aliases":                   types.ListNull(types.StringType),
		"ra": types.ObjectValueMust(networkIPv6RAAttrTypes(), map[string]attr.Value{
			"enabled":            types.BoolValue(true),
			"priority":           types.StringValue("high"),
			"preferred_lifetime": timetypes.NewGoDurationValueFromStringMust("4h"),
			"valid_lifetime":     timetypes.NewGoDurationValueFromStringMust("24h"),
		}),
		"pd": types.ObjectValueMust(networkIPv6PDAttrTypes(), map[string]attr.Value{
			"interface":             types.StringValue("wan"),
			"prefixid":              types.StringValue("1a"),
			"start":                 types.StringValue("::2"),
			"stop":                  types.StringValue("::7d1"),
			"auto_prefixid_enabled": types.BoolValue(false),
		}),
	})
	model.DhcpServer = dhcpServer(
		option(true, strList("10.0.1.53", "10.0.1.54")),
		option(true, strList("10.0.1.123")),
	)
	model.DhcpV6Server = dhcpV6(types.ObjectValueMust(
		dhcpV6DNSModel{}.AttributeTypes(),
		map[string]attr.Value{
			"auto":    types.BoolValue(false),
			"servers": strList("2001:4860:4860::8888"),
		},
	))

	api, diags := r.modelToNetwork(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork: %v", diags)
	}
	deref := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	if deref(api.IPV6InterfaceType) != "pd" || deref(api.IPV6ClientAddressAssignment) != "slaac" ||
		api.IPV6Subnet != nil {
		t.Errorf("ipv6 top level: %q %q %v",
			deref(api.IPV6InterfaceType), deref(api.IPV6ClientAddressAssignment), api.IPV6Subnet)
	}
	if !api.IPV6RaEnabled || deref(api.IPV6RaPriority) != "high" ||
		api.IPV6RaPreferredLifetime == nil || *api.IPV6RaPreferredLifetime != 14400 ||
		api.IPV6RaValidLifetime == nil || *api.IPV6RaValidLifetime != 86400 {
		t.Errorf("ipv6.ra: %v %q %v %v", api.IPV6RaEnabled, deref(api.IPV6RaPriority),
			api.IPV6RaPreferredLifetime, api.IPV6RaValidLifetime)
	}
	if deref(api.IPV6PDInterface) != "wan" || api.IPV6PDPrefixid != "1a" ||
		deref(api.IPV6PDStart) != "::2" || deref(api.IPV6PDStop) != "::7d1" ||
		api.IPV6PDAutoPrefixidEnabled {
		t.Errorf("ipv6.pd: %q %q %q %q %v", deref(api.IPV6PDInterface), api.IPV6PDPrefixid,
			deref(api.IPV6PDStart), deref(api.IPV6PDStop), api.IPV6PDAutoPrefixidEnabled)
	}
	if !api.DHCPDDNSEnabled ||
		derefString(api.DHCPDDNS1) != "10.0.1.53" ||
		derefString(api.DHCPDDNS2) != "10.0.1.54" ||
		derefString(api.DHCPDDNS3) != "" ||
		derefString(api.DHCPDDNS4) != "" {
		t.Errorf("dhcp_server.dns: %v %q %q %q %q", api.DHCPDDNSEnabled,
			derefString(api.DHCPDDNS1), derefString(api.DHCPDDNS2),
			derefString(api.DHCPDDNS3), derefString(api.DHCPDDNS4))
	}
	if !api.DHCPDNtpEnabled || deref(api.DHCPDNtp1) != "10.0.1.123" || deref(api.DHCPDNtp2) != "" {
		t.Errorf("dhcp_server.ntp: %v %q %q", api.DHCPDNtpEnabled,
			deref(api.DHCPDNtp1), deref(api.DHCPDNtp2))
	}
	if api.DHCPDV6DNSAuto || deref(api.DHCPDV6DNS1) != "2001:4860:4860::8888" ||
		deref(api.DHCPDV6DNS2) != "" {
		t.Errorf("dhcp_v6_server.dns: %v %q %q", api.DHCPDV6DNSAuto,
			deref(api.DHCPDV6DNS1), deref(api.DHCPDV6DNS2))
	}

	// Read back: every group is rebuilt from the API response.
	api.ID = "net-1"
	api.Purpose = unifi.PurposeCorporate
	var back networkResourceModel
	if d := r.networkToModel(ctx, api, &back, "default", model); d.HasError() {
		t.Fatalf("networkToModel: %v", d)
	}
	v6 := back.IPv6.Attributes()
	if attrAs[types.String](t, v6["interface_type"]).ValueString() != "pd" ||
		attrAs[types.String](t, v6["client_address_assignment"]).ValueString() != "slaac" ||
		!v6["aliases"].IsNull() {
		t.Errorf("ipv6 read back = %v", back.IPv6)
	}
	ra := attrAs[types.Object](t, v6["ra"]).Attributes()
	if !attrAs[types.Bool](t, ra["enabled"]).ValueBool() ||
		attrAs[types.String](t, ra["priority"]).ValueString() != "high" ||
		durationOf(t, ra["preferred_lifetime"]) != 4*time.Hour {
		t.Errorf("ipv6.ra read back = %v", ra)
	}
	pd := attrAs[types.Object](t, v6["pd"]).Attributes()
	if attrAs[types.String](t, pd["interface"]).ValueString() != "wan" ||
		attrAs[types.String](t, pd["prefixid"]).ValueString() != "1a" ||
		attrAs[types.String](t, pd["stop"]).ValueString() != "::7d1" {
		t.Errorf("ipv6.pd read back = %v", pd)
	}
	dhcp := back.DhcpServer.Attributes()
	dns := attrAs[types.Object](t, dhcp["dns"]).Attributes()
	if !attrAs[types.Bool](t, dns["enabled"]).ValueBool() ||
		len(attrAs[types.List](t, dns["servers"]).Elements()) != 2 {
		t.Errorf("dhcp_server.dns read back = %v", dns)
	}
	ntp := attrAs[types.Object](t, dhcp["ntp"]).Attributes()
	if !attrAs[types.Bool](t, ntp["enabled"]).ValueBool() ||
		len(attrAs[types.List](t, ntp["servers"]).Elements()) != 1 {
		t.Errorf("dhcp_server.ntp read back = %v", ntp)
	}
	v6dns := attrAs[types.Object](t, back.DhcpV6Server.Attributes()["dns"]).Attributes()
	if attrAs[types.Bool](t, v6dns["auto"]).ValueBool() ||
		len(attrAs[types.List](t, v6dns["servers"]).Elements()) != 1 {
		t.Errorf("dhcp_v6_server.dns read back = %v", v6dns)
	}

	// A null group contributes what the unset flat attributes did: nothing
	// for ipv6, and disabled-with-cleared-slots for the DHCP options.
	unset := base()
	unset.IPv6 = types.ObjectNull(networkIPv6AttrTypes())
	unset.DhcpServer = dhcpServer(
		types.ObjectNull(dhcpServerOptionModel{}.AttributeTypes()),
		types.ObjectNull(dhcpServerOptionModel{}.AttributeTypes()),
	)
	unset.DhcpV6Server = dhcpV6(types.ObjectNull(dhcpV6DNSModel{}.AttributeTypes()))
	api, diags = r.modelToNetwork(ctx, unset)
	if diags.HasError() {
		t.Fatalf("modelToNetwork (null groups): %v", diags)
	}
	if api.IPV6InterfaceType != nil || api.IPV6RaEnabled || api.IPV6RaPriority != nil ||
		api.IPV6PDPrefixid != "" || api.IPV6PDStart != nil {
		t.Errorf("null ipv6 must contribute nothing: %+v", api)
	}
	if api.DHCPDDNSEnabled || derefString(api.DHCPDDNS1) != "" || api.DHCPDNtpEnabled ||
		deref(api.DHCPDNtp1) != "" || deref(api.DHCPDNtp2) != "" {
		t.Errorf("null dhcp_server.dns/ntp must clear the slots: %+v", api)
	}
	if api.DHCPDV6DNSAuto || deref(api.DHCPDV6DNS1) != "" || deref(api.DHCPDV6DNS4) != "" {
		t.Errorf("null dhcp_v6_server.dns must clear the slots: %+v", api)
	}
}

// TestNetworkIPv6_vlanOnlyPreservesPrevious guards the vlan-only read path:
// the controller omits the IPv6 fields, so the previous object is preserved
// leaf by leaf, unknown leaves (Computed + UseStateForUnknown on Create)
// resolve from the controller, an absent interface_type normalizes to "none"
// and a null or unknown previous object behaves like null or unknown leaves.
func TestNetworkIPv6_vlanOnlyPreservesPrevious(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	network := &unifi.Network{
		ID:                          "net-vo",
		Name:                        stringPtr("VLAN only"),
		Purpose:                     unifi.PurposeVLANOnly,
		Enabled:                     true,
		IPV6ClientAddressAssignment: stringPtr("slaac"),
		IPV6RaEnabled:               false,
		IPV6RaPriority:              stringPtr("medium"),
		IPV6PDStart:                 stringPtr("::2"),
	}
	previousModel := func(ipv6 types.Object) *networkResourceModel {
		return &networkResourceModel{
			IPv6:         ipv6,
			DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases: types.ListNull(types.StringType),
		}
	}
	read := func(t *testing.T, previous types.Object) map[string]attr.Value {
		t.Helper()
		var model networkResourceModel
		if d := r.networkToModel(
			ctx,
			network,
			&model,
			"default",
			previousModel(previous),
		); d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if model.IPv6.IsNull() || model.IPv6.IsUnknown() {
			t.Fatalf("ipv6 = %v, want a known object", model.IPv6)
		}
		return model.IPv6.Attributes()
	}

	t.Run("plan shape: unknown leaves resolve, known ones are kept", func(t *testing.T) {
		v6 := read(t, networkIPv6PlanShape())
		if attrAs[types.String](t, v6["interface_type"]).ValueString() != "none" {
			t.Errorf("interface_type = %v, want none", v6["interface_type"])
		}
		if attrAs[types.String](t, v6["client_address_assignment"]).ValueString() != "slaac" {
			t.Errorf("client_address_assignment = %v, want slaac (from controller)",
				v6["client_address_assignment"])
		}
		if !v6["static_subnet"].IsNull() || !v6["aliases"].IsNull() {
			t.Errorf("static_subnet/aliases = %v/%v, want null", v6["static_subnet"], v6["aliases"])
		}
		ra := attrAs[types.Object](t, v6["ra"]).Attributes()
		if attrAs[types.Bool](t, ra["enabled"]).ValueBool() ||
			attrAs[types.String](t, ra["priority"]).ValueString() != "medium" ||
			!ra["preferred_lifetime"].IsNull() {
			t.Errorf("ra = %v, want controller values", ra)
		}
		pd := attrAs[types.Object](t, v6["pd"]).Attributes()
		if !pd["interface"].IsNull() ||
			attrAs[types.String](t, pd["start"]).ValueString() != "::2" ||
			!pd["stop"].IsNull() ||
			attrAs[types.Bool](t, pd["auto_prefixid_enabled"]).ValueBool() {
			t.Errorf("pd = %v, want null interface and controller start/stop", pd)
		}
		for _, v := range v6 {
			if v.IsUnknown() {
				t.Errorf("unknown leaf survived the read: %v", v6)
			}
		}
	})

	t.Run("configured values are preserved over the controller", func(t *testing.T) {
		prev := types.ObjectValueMust(networkIPv6AttrTypes(), map[string]attr.Value{
			"interface_type":            types.StringValue("static"),
			"client_address_assignment": types.StringValue("dhcpv6"),
			"static_subnet":             types.StringValue("fd00::1/64"),
			"aliases":                   types.ListNull(types.StringType),
			"ra": types.ObjectValueMust(networkIPv6RAAttrTypes(), map[string]attr.Value{
				"enabled":            types.BoolValue(true),
				"priority":           types.StringUnknown(),
				"preferred_lifetime": timetypes.NewGoDurationValueFromStringMust("4h"),
				"valid_lifetime":     timetypes.NewGoDurationNull(),
			}),
			"pd": networkIPv6PDPlanShape(),
		})
		v6 := read(t, prev)
		if attrAs[types.String](t, v6["interface_type"]).ValueString() != "static" ||
			attrAs[types.String](t, v6["client_address_assignment"]).ValueString() != "dhcpv6" ||
			attrAs[types.String](t, v6["static_subnet"]).ValueString() != "fd00::1/64" {
			t.Errorf("configured leaves not preserved: %v", v6)
		}
		ra := attrAs[types.Object](t, v6["ra"]).Attributes()
		if !attrAs[types.Bool](t, ra["enabled"]).ValueBool() ||
			attrAs[types.String](t, ra["priority"]).ValueString() != "medium" ||
			durationOf(t, ra["preferred_lifetime"]) != 4*time.Hour ||
			!ra["valid_lifetime"].IsNull() {
			t.Errorf("ra = %v, want configured leaves kept and unknown priority resolved", ra)
		}
	})

	t.Run("null previous (import) yields none and null leaves", func(t *testing.T) {
		v6 := read(t, types.ObjectNull(networkIPv6AttrTypes()))
		if attrAs[types.String](t, v6["interface_type"]).ValueString() != "none" {
			t.Errorf("interface_type = %v, want none", v6["interface_type"])
		}
		if !v6["client_address_assignment"].IsNull() || !v6["ra"].IsNull() || !v6["pd"].IsNull() {
			t.Errorf("null previous must keep null leaves: %v", v6)
		}
	})

	t.Run("unknown previous resolves from the controller", func(t *testing.T) {
		v6 := read(t, types.ObjectUnknown(networkIPv6AttrTypes()))
		if attrAs[types.String](t, v6["interface_type"]).ValueString() != "none" {
			t.Errorf("interface_type = %v, want none", v6["interface_type"])
		}
		if attrAs[types.String](t, v6["client_address_assignment"]).ValueString() != "slaac" {
			t.Errorf("client_address_assignment = %v, want slaac", v6["client_address_assignment"])
		}
		ra := attrAs[types.Object](t, v6["ra"]).Attributes()
		if attrAs[types.String](t, ra["priority"]).ValueString() != "medium" {
			t.Errorf("ra.priority = %v, want medium", ra["priority"])
		}
	})
}

// TestNetworkDataSource_nestedGroups checks the data source's ipv6, wan and
// dhcp option objects are populated from the API response.
func TestNetworkDataSource_nestedGroups(t *testing.T) {
	ctx := context.Background()
	d := &networkDataSource{}

	network := &unifi.Network{
		ID:                      "net-ds",
		Name:                    stringPtr("wan"),
		Purpose:                 "wan",
		IPV6InterfaceType:       stringPtr("pd"),
		IPV6RaEnabled:           true,
		IPV6RaPriority:          stringPtr("high"),
		IPV6RaPreferredLifetime: func() *int64 { v := int64(14400); return &v }(),
		IPV6PDInterface:         stringPtr("wan"),
		IPV6PDPrefixid:          "",
		IPV6PDStart:             stringPtr("::2"),
		DHCPDDNSEnabled:         true,
		DHCPDDNS1:               new("10.0.0.53"),
		DHCPDNtp1:               stringPtr("10.0.0.123"),
		DHCPDV6DNSAuto:          true,
		WANDNS1:                 stringPtr("1.1.1.1"),
		WANDNS3:                 "9.9.9.9",
		WANGateway:              stringPtr("203.0.113.1"),
		WANGatewayV6:            "",
		WANIP:                   stringPtr("203.0.113.10"),
		WANNetworkGroup:         stringPtr("WAN"),
		WANType:                 stringPtr("static"),
		WANUsername:             "user",
	}

	var model networkDataSourceModel
	var diags diag.Diagnostics
	d.setDataSourceData(ctx, &diags, network, &model, "default")
	if diags.HasError() {
		t.Fatalf("setDataSourceData: %v", diags)
	}

	v6 := model.IPv6.Attributes()
	if attrAs[types.String](t, v6["interface_type"]).ValueString() != "pd" ||
		!v6["aliases"].IsNull() {
		t.Errorf("ipv6 = %v", model.IPv6)
	}
	ra := attrAs[types.Object](t, v6["ra"]).Attributes()
	if !attrAs[types.Bool](t, ra["enabled"]).ValueBool() ||
		attrAs[types.String](t, ra["priority"]).ValueString() != "high" ||
		durationOf(t, ra["preferred_lifetime"]) != 4*time.Hour ||
		!ra["valid_lifetime"].IsNull() {
		t.Errorf("ipv6.ra = %v", ra)
	}
	pd := attrAs[types.Object](t, v6["pd"]).Attributes()
	if attrAs[types.String](t, pd["interface"]).ValueString() != "wan" ||
		!pd["prefixid"].IsNull() ||
		attrAs[types.String](t, pd["start"]).ValueString() != "::2" {
		t.Errorf("ipv6.pd = %v", pd)
	}

	dhcp := model.DhcpServer.Attributes()
	dns := attrAs[types.Object](t, dhcp["dns"]).Attributes()
	if !attrAs[types.Bool](t, dns["enabled"]).ValueBool() ||
		len(attrAs[types.List](t, dns["servers"]).Elements()) != 1 {
		t.Errorf("dhcp_server.dns = %v", dns)
	}
	ntp := attrAs[types.Object](t, dhcp["ntp"]).Attributes()
	if attrAs[types.Bool](t, ntp["enabled"]).ValueBool() ||
		len(attrAs[types.List](t, ntp["servers"]).Elements()) != 1 {
		t.Errorf("dhcp_server.ntp = %v", ntp)
	}
	v6dns := attrAs[types.Object](t, model.DhcpV6Server.Attributes()["dns"]).Attributes()
	if !attrAs[types.Bool](t, v6dns["auto"]).ValueBool() || !v6dns["servers"].IsNull() {
		t.Errorf("dhcp_v6_server.dns = %v", v6dns)
	}

	wan := model.Wan.Attributes()
	if len(attrAs[types.List](t, wan["dns"]).Elements()) != 2 ||
		attrAs[types.String](t, wan["gateway"]).ValueString() != "203.0.113.1" ||
		!wan["gateway_v6"].IsNull() ||
		attrAs[types.String](t, wan["ip"]).ValueString() != "203.0.113.10" ||
		!wan["netmask"].IsNull() ||
		attrAs[types.String](t, wan["network_group"]).ValueString() != "WAN" ||
		attrAs[types.String](t, wan["type"]).ValueString() != "static" ||
		!wan["type_v6"].IsNull() ||
		attrAs[types.String](t, wan["username"]).ValueString() != "user" ||
		!wan["egress_qos"].IsNull() {
		t.Errorf("wan = %v", model.Wan)
	}
}
