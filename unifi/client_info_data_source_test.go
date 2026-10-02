package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
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
