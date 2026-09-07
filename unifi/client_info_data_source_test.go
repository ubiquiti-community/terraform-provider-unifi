package unifi

import (
	"context"
	"reflect"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestNewClientInfoDataSource(t *testing.T) {
	d := NewClientInfoDataSource()
	if d == nil {
		t.Fatal("NewClientInfoDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_clientInfoDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_client_info"},
		{"test", "test_client_info"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &clientInfoDataSource{}
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

func Test_clientInfoDataSource_Schema(t *testing.T) {
	d := &clientInfoDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "mac"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
	assertClientInfoNestedGroups(t, resp.Schema.Attributes, true)
}

// Test_clientInfoDataSource_modelMatchesSchema guards the tfsdk tags of
// clientInfoDataSourceModel against the shared models.Attributes() schema, so
// a nested group renamed in one place cannot drift from the other.
func Test_clientInfoDataSource_modelMatchesSchema(t *testing.T) {
	d := &clientInfoDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)

	tags := map[string]bool{}
	rt := reflect.TypeOf(clientInfoDataSourceModel{})
	for i := 0; i < rt.NumField(); i++ {
		tags[rt.Field(i).Tag.Get("tfsdk")] = true
	}
	for name := range resp.Schema.Attributes {
		if !tags[name] {
			t.Errorf("schema attribute %q has no model field", name)
		}
	}
	for name := range tags {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("model field %q has no schema attribute", name)
		}
	}
}

// assertClientInfoNestedGroups checks the network/last_uplink/
// last_connection_network/tx/rx nested objects are present as Computed
// SingleNestedAttributes with the expected leaves and that no flat, prefixed
// attribute remains. withRemotePort selects the client_info shape of
// last_uplink (mac, name, remote_port) over the client_list one (mac, name).
func assertClientInfoNestedGroups(
	t *testing.T,
	attrs map[string]schema.Attribute,
	withRemotePort bool,
) {
	t.Helper()
	lastUplink := []string{"mac", "name"}
	if withRemotePort {
		lastUplink = append(lastUplink, "remote_port")
	}
	for name, leaves := range map[string][]string{
		"network":                 {"id", "name"},
		"last_uplink":             lastUplink,
		"last_connection_network": {"id", "name"},
		"tx":                      {"rate", "bytes"},
		"rx":                      {"rate", "bytes"},
	} {
		a, ok := attrs[name]
		if !ok {
			t.Errorf("missing nested attribute %q", name)
			continue
		}
		nested, ok := a.(schema.SingleNestedAttribute)
		if !ok {
			t.Errorf("%s: is %T, want schema.SingleNestedAttribute", name, a)
			continue
		}
		if !nested.Computed || nested.Optional || nested.Required {
			t.Errorf("%s: want Computed-only object", name)
		}
		if len(nested.Attributes) != len(leaves) {
			t.Errorf("%s: has %d leaves, want %d", name, len(nested.Attributes), len(leaves))
		}
		for _, leaf := range leaves {
			la, ok := nested.Attributes[leaf]
			if !ok {
				t.Errorf("%s.%s: missing leaf", name, leaf)
				continue
			}
			if !la.IsComputed() || la.IsOptional() || la.IsRequired() {
				t.Errorf("%s.%s: want Computed-only leaf", name, leaf)
			}
		}
	}
	for _, old := range []string{
		"network_id", "network_name",
		"last_uplink_mac", "last_uplink_name", "last_uplink_remote_port",
		"last_connection_network_id", "last_connection_network_name",
		"tx_rate", "tx_bytes", "rx_rate", "rx_bytes",
	} {
		if _, ok := attrs[old]; ok {
			t.Errorf("flat attribute %q still present", old)
		}
	}
}

func Test_clientInfoDataSource_Configure(t *testing.T) {
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
			d := &clientInfoDataSource{}
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
