package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestNewPortProfileDataSource(t *testing.T) {
	d := NewPortProfileDataSource()
	if d == nil {
		t.Fatal("NewPortProfileDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_portProfileDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_port_profile"},
		{"test", "test_port_profile"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &portProfileDataSource{}
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

func Test_portProfileDataSource_Schema(t *testing.T) {
	d := &portProfileDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "name", "forward"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_portProfileDataSource_Configure(t *testing.T) {
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
			d := &portProfileDataSource{}
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
