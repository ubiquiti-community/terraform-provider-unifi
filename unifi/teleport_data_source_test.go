package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

func TestNewTeleportDataSource(t *testing.T) {
	d := NewTeleportDataSource()
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_teleportDataSource_Metadata(t *testing.T) {
	resp := &fwdatasource.MetadataResponse{}
	(&teleportDataSource{}).Metadata(
		context.Background(),
		fwdatasource.MetadataRequest{ProviderTypeName: "unifi"},
		resp,
	)
	if resp.TypeName != "unifi_teleport" {
		t.Errorf("TypeName = %q, want unifi_teleport", resp.TypeName)
	}
}

func Test_teleportDataSource_Schema(t *testing.T) {
	ctx := context.Background()
	resp := &fwdatasource.SchemaResponse{}
	(&teleportDataSource{}).Schema(ctx, fwdatasource.SchemaRequest{}, resp)
	if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("schema is invalid: %v", diags)
	}
	for _, attr := range []string{"id", "site", "enabled", "subnet_cidr", "timeouts"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_teleportDataSource_Configure(t *testing.T) {
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
			resp := &fwdatasource.ConfigureResponse{}
			(&teleportDataSource{}).Configure(
				context.Background(),
				fwdatasource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

func Test_teleportToDataSourceModel(t *testing.T) {
	t.Run("maps a configured subnet", func(t *testing.T) {
		var got teleportDataSourceModel
		teleportToDataSourceModel(
			&settings.Teleport{Enabled: true, SubnetCidr: "192.168.100.0/24"},
			&got,
			"default",
		)
		if got.ID.ValueString() != "default" || got.Site.ValueString() != "default" {
			t.Errorf("id/site = %q/%q, want default", got.ID.ValueString(), got.Site.ValueString())
		}
		if !got.Enabled.ValueBool() || got.SubnetCidr.ValueString() != "192.168.100.0/24" {
			t.Errorf("model = %+v", got)
		}
	})

	t.Run("an empty subnet is null", func(t *testing.T) {
		var got teleportDataSourceModel
		teleportToDataSourceModel(&settings.Teleport{}, &got, "default")
		if got.Enabled.ValueBool() {
			t.Error("Enabled = true, want false")
		}
		if !got.SubnetCidr.IsNull() {
			t.Errorf("SubnetCidr = %v, want null", got.SubnetCidr)
		}
	})
}
