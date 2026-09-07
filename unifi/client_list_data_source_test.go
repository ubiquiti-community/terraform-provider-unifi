package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gounifi "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/models"
)

func TestAccClientListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.test",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.test",
						"site",
						"default",
					),
				),
			},
		},
	})
}

func TestAccClientListDataSource_filtered(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_wired(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.wired",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.wired",
						"site",
						"default",
					),
				),
			},
			{
				Config: testAccClientListDataSourceConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.blocked",
						"clients.#",
					),
				),
			},
		},
	})
}

func testAccClientListDataSourceConfig_basic() string {
	return `
data "unifi_client_list" "test" {
}
`
}

func testAccClientListDataSourceConfig_wired() string {
	return `
data "unifi_client_list" "wired" {
  wired = true
}
`
}

func testAccClientListDataSourceConfig_blocked() string {
	return `
data "unifi_client_list" "blocked" {
  blocked = false
}
`
}

func TestNewClientListDataSource(t *testing.T) {
	d := NewClientListDataSource()
	if d == nil {
		t.Fatal("NewClientListDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_clientListEntryAttrTypes(t *testing.T) {
	got := clientListEntryAttrTypes()
	for _, key := range []string{"id", "mac", "name", "ip", "blocked", "is_wired", "status", "uptime", "first_seen", "last_seen"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	if got["id"] != types.StringType {
		t.Errorf("id type = %T, want StringType", got["id"])
	}
	if got["blocked"] != types.BoolType {
		t.Errorf("blocked type = %T, want BoolType", got["blocked"])
	}
	if got["uptime"] != types.Int64Type {
		t.Errorf("uptime type = %T, want Int64Type", got["uptime"])
	}
	for name, want := range map[string]attr.Type{
		"network":                 types.ObjectType{AttrTypes: models.ClientNetworkAttrTypes()},
		"last_uplink":             types.ObjectType{AttrTypes: clientListLastUplinkAttrTypes()},
		"last_connection_network": types.ObjectType{AttrTypes: models.ClientNetworkAttrTypes()},
		"tx":                      types.ObjectType{AttrTypes: models.ClientTrafficAttrTypes()},
		"rx":                      types.ObjectType{AttrTypes: models.ClientTrafficAttrTypes()},
	} {
		if !got[name].Equal(want) {
			t.Errorf("%s type = %s, want %s", name, got[name], want)
		}
	}
	for _, old := range []string{
		"network_id", "network_name", "last_uplink_mac", "last_uplink_name",
		"last_connection_network_id", "last_connection_network_name",
		"tx_rate", "tx_bytes", "rx_rate", "rx_bytes",
	} {
		if _, ok := got[old]; ok {
			t.Errorf("flat attribute %q still present", old)
		}
	}
	// Every attr type must have a schema attribute of the same type.
	attrs := clientListEntrySchemaAttributes()
	if len(attrs) != len(got) {
		t.Errorf("schema has %d attributes, attr types %d", len(attrs), len(got))
	}
	for name, want := range got {
		a, ok := attrs[name]
		if !ok {
			t.Errorf("attr type %q has no schema attribute", name)
			continue
		}
		if !a.GetType().Equal(want) {
			t.Errorf("%s: schema type %s != attr type %s", name, a.GetType(), want)
		}
	}
}

func Test_clientListEntrySchemaAttributes(t *testing.T) {
	got := clientListEntrySchemaAttributes()
	for _, key := range []string{"id", "mac", "name", "ip", "blocked", "is_wired", "status"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing attribute %q", key)
		}
	}
	assertClientInfoNestedGroups(t, got, false)
}

func Test_clientListDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_client_list"},
		{"test", "test_client_list"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &clientListDataSource{}
			resp := &fwdatasource.MetadataResponse{}
			d.Metadata(
				context.Background(),
				fwdatasource.MetadataRequest{ProviderTypeName: tt.providerTypeName},
				resp,
			)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_clientListDataSource_Schema(t *testing.T) {
	d := &clientListDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"site", "group", "wired", "blocked", "oui", "clients"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_clientListDataSource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		data      any
		wantError bool
	}{
		{"nil provider data", nil, false},
		{"wrong type", "wrong", true},
		{"correct client type", &Client{Site: "default"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &clientListDataSource{}
			resp := &fwdatasource.ConfigureResponse{}
			d.Configure(
				context.Background(),
				fwdatasource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error in diagnostics")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

func Test_clientListEntryValues(t *testing.T) {
	ctx := context.Background()
	c := &gounifi.Client{
		ID:  "abc123",
		MAC: "aa:bb:cc:dd:ee:ff",
	}
	got, diags := clientListEntryValues(ctx, c, nil)
	if diags.HasError() {
		t.Fatalf("clientListEntryValues: %v", diags)
	}
	if got["id"] != types.StringValue("abc123") {
		t.Errorf("id = %v, want %q", got["id"], "abc123")
	}
	if got["mac"] != types.StringValue("aa:bb:cc:dd:ee:ff") {
		t.Errorf("mac = %v, want %q", got["mac"], "aa:bb:cc:dd:ee:ff")
	}
}

// Test_clientListEntryValues_nestedGroups checks the network, last_uplink,
// last_connection_network, tx and rx objects are populated from ClientInfo and
// that the value map satisfies clientListEntryAttrTypes().
func Test_clientListEntryValues_nestedGroups(t *testing.T) {
	ctx := context.Background()
	txRate, txBytes := int64(1000), int64(2000)
	rxRate, rxBytes := int64(3000), int64(4000)
	c := &gounifi.Client{
		ID:                       "abc123",
		MAC:                      "aa:bb:cc:dd:ee:ff",
		NetworkID:                "net-1",
		VirtualNetworkOverrideID: "vnet-1",
	}
	info := &gounifi.ClientInfo{
		NetworkName:               "LAN",
		LastUplinkMac:             "11:22:33:44:55:66",
		LastUplinkName:            "sw-1",
		LastConnectionNetworkId:   "net-2",
		LastConnectionNetworkName: "IoT",
		TxRate:                    &txRate,
		TxBytes:                   &txBytes,
		RxRate:                    &rxRate,
		RxBytes:                   &rxBytes,
	}

	got, diags := clientListEntryValues(ctx, c, info)
	if diags.HasError() {
		t.Fatalf("clientListEntryValues: %v", diags)
	}
	if _, d := types.ObjectValue(clientListEntryAttrTypes(), got); d.HasError() {
		t.Fatalf("value map does not satisfy clientListEntryAttrTypes(): %v", d)
	}

	leaf := func(name, leaf string) attr.Value {
		t.Helper()
		obj := attrAs[types.Object](t, got[name])
		v, ok := obj.Attributes()[leaf]
		if !ok {
			t.Fatalf("%s.%s missing", name, leaf)
		}
		return v
	}
	// network.id prefers the virtual network override, as network_id did.
	if v := leaf("network", "id"); !v.Equal(types.StringValue("vnet-1")) {
		t.Errorf("network.id = %v, want vnet-1", v)
	}
	if v := leaf("network", "name"); !v.Equal(types.StringValue("LAN")) {
		t.Errorf("network.name = %v, want LAN", v)
	}
	if v := leaf("last_uplink", "mac"); !v.Equal(types.StringValue("11:22:33:44:55:66")) {
		t.Errorf("last_uplink.mac = %v", v)
	}
	if v := leaf("last_uplink", "name"); !v.Equal(types.StringValue("sw-1")) {
		t.Errorf("last_uplink.name = %v", v)
	}
	if v := leaf("last_connection_network", "id"); !v.Equal(types.StringValue("net-2")) {
		t.Errorf("last_connection_network.id = %v", v)
	}
	if v := leaf("last_connection_network", "name"); !v.Equal(types.StringValue("IoT")) {
		t.Errorf("last_connection_network.name = %v", v)
	}
	if v := leaf("tx", "rate"); !v.Equal(types.Int64Value(1000)) {
		t.Errorf("tx.rate = %v", v)
	}
	if v := leaf("tx", "bytes"); !v.Equal(types.Int64Value(2000)) {
		t.Errorf("tx.bytes = %v", v)
	}
	if v := leaf("rx", "rate"); !v.Equal(types.Int64Value(3000)) {
		t.Errorf("rx.rate = %v", v)
	}
	if v := leaf("rx", "bytes"); !v.Equal(types.Int64Value(4000)) {
		t.Errorf("rx.bytes = %v", v)
	}
}

// Test_clientListEntryValues_noInfo checks that without ClientInfo the nested
// objects are still known but every ClientInfo-sourced leaf is null, while
// network.id still comes from the Client record.
func Test_clientListEntryValues_noInfo(t *testing.T) {
	ctx := context.Background()
	c := &gounifi.Client{ID: "abc123", MAC: "aa:bb:cc:dd:ee:ff", NetworkID: "net-1"}

	got, diags := clientListEntryValues(ctx, c, nil)
	if diags.HasError() {
		t.Fatalf("clientListEntryValues: %v", diags)
	}
	if _, d := types.ObjectValue(clientListEntryAttrTypes(), got); d.HasError() {
		t.Fatalf("value map does not satisfy clientListEntryAttrTypes(): %v", d)
	}

	network := attrAs[types.Object](t, got["network"])
	if v := network.Attributes()["id"]; !v.Equal(types.StringValue("net-1")) {
		t.Errorf("network.id = %v, want net-1", v)
	}
	if v := network.Attributes()["name"]; !v.IsNull() {
		t.Errorf("network.name = %v, want null", v)
	}
	for _, name := range []string{"last_uplink", "last_connection_network", "tx", "rx"} {
		obj := attrAs[types.Object](t, got[name])
		if obj.IsNull() || obj.IsUnknown() {
			t.Errorf("%s: object should be known, got %v", name, obj)
			continue
		}
		for leaf, v := range obj.Attributes() {
			if !v.IsNull() {
				t.Errorf("%s.%s = %v, want null", name, leaf, v)
			}
		}
	}
}

func Test_networkIDValue(t *testing.T) {
	t.Run("VirtualNetworkOverrideID wins", func(t *testing.T) {
		c := &gounifi.Client{NetworkID: "net1", VirtualNetworkOverrideID: "vnet1"}
		got := networkIDValue(c)
		if got.ValueString() != "vnet1" {
			t.Errorf("got %q, want %q", got.ValueString(), "vnet1")
		}
	})
	t.Run("falls back to NetworkID", func(t *testing.T) {
		c := &gounifi.Client{NetworkID: "net1"}
		got := networkIDValue(c)
		if got.ValueString() != "net1" {
			t.Errorf("got %q, want %q", got.ValueString(), "net1")
		}
	})
}

func Test_stringSliceToList(t *testing.T) {
	t.Run("nil slice returns null list", func(t *testing.T) {
		got := stringSliceToList(nil)
		if !got.IsNull() {
			t.Error("expected null list for nil slice")
		}
	})
	t.Run("empty slice returns null list", func(t *testing.T) {
		got := stringSliceToList([]string{})
		if !got.IsNull() {
			t.Error("expected null list for empty slice")
		}
	})
	t.Run("populated slice returns list value", func(t *testing.T) {
		got := stringSliceToList([]string{"a", "b"})
		if got.IsNull() || got.IsUnknown() {
			t.Error("expected non-null list")
		}
		if len(got.Elements()) != 2 {
			t.Errorf("len = %d, want 2", len(got.Elements()))
		}
	})
}

func Test_int64PointerValueOrNull(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := int64PointerValueOrNull(nil)
		if !got.IsNull() {
			t.Error("expected null Int64 for nil pointer")
		}
	})
	t.Run("non-nil returns value", func(t *testing.T) {
		v := int64(42)
		got := int64PointerValueOrNull(&v)
		if got.ValueInt64() != 42 {
			t.Errorf("got %d, want 42", got.ValueInt64())
		}
	})
}
