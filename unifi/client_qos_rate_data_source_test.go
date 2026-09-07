package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func TestNewClientQosRateDataSource(t *testing.T) {
	d := NewClientQosRateDataSource()
	if d == nil {
		t.Fatal("NewClientQosRateDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_clientQosRateDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_client_qos_rate"},
		{"test", "test_client_qos_rate"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &clientQosRateDataSource{}
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

func Test_clientQosRateDataSource_Schema(t *testing.T) {
	d := &clientQosRateDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "name", "qos_rate"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
	rate, ok := resp.Schema.Attributes["qos_rate"].(dsschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf(
			"qos_rate is %T, want SingleNestedAttribute",
			resp.Schema.Attributes["qos_rate"],
		)
	}
	if !rate.Computed || rate.Optional || rate.Required {
		t.Errorf("qos_rate should be Computed only: %+v", rate)
	}
	for _, leaf := range []string{"max_down", "max_up"} {
		if _, ok := rate.Attributes[leaf]; !ok {
			t.Errorf("missing nested attribute qos_rate.%s", leaf)
		}
	}
}

func Test_clientQosRateDataSource_Configure(t *testing.T) {
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
			d := &clientQosRateDataSource{}
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
