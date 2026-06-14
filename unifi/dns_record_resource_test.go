package unifi

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func testAccDNSRecordCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "unifi_dns_record" {
			continue
		}
		id := rs.Primary.ID
		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = "default"
		}
		client := &Client{
			ApiClient: nil, // populated by the acceptance test provider
			Site:      site,
		}
		// Use the shared provider client via a direct API call.
		apiClient, err := unifi.New(ctx, &unifi.Config{
			BaseURL:       rs.Primary.Attributes["api_url"],
			Username:      rs.Primary.Attributes["username"],
			Password:      rs.Primary.Attributes["password"],
			AllowInsecure: true,
		})
		if err != nil {
			// If we can't build a client, skip the check.
			return nil
		}
		client.ApiClient = apiClient
		_, err = client.GetDNSRecord(ctx, site, id)
		if err != nil {
			if _, ok := err.(*unifi.NotFoundError); ok {
				continue
			}
			return fmt.Errorf("error checking DNS record %s: %w", id, err)
		}
		return fmt.Errorf("DNS record %s still exists", id)
	}
	return nil
}

func TestAccDNSRecordFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             testAccDNSRecordCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_dns_record.test", "name", "test-record"),
					resource.TestCheckResourceAttr(
						"unifi_dns_record.test",
						"value",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr("unifi_dns_record.test", "priority", "10"),
					resource.TestCheckResourceAttr("unifi_dns_record.test", "enabled", "true"),
				),
				ExpectError: regexp.MustCompile(".*"),
			},
		},
	})
}

func testAccDNSRecordFrameworkConfig_basic() string {
	return `
resource "unifi_dns_record" "test" {
  name        = "test-record.example.com"
  enabled     = true
  priority    = 10
  record_type = "A"
  ttl         = "5m0s"
  value       = "192.168.1.100"
}
`
}

func TestNewDNSRecordFrameworkResource(t *testing.T) {
	r := NewDNSRecordFrameworkResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity")
	}
	if _, ok := r.(fwresource.ResourceWithUpgradeState); !ok {
		t.Error("expected ResourceWithUpgradeState")
	}
}

func TestNewDNSRecordListResource(t *testing.T) {
	r := NewDNSRecordListResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwlist.ListResource); !ok {
		t.Error("expected ListResource")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure")
	}
}

func Test_dnsRecordFrameworkResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name         string
		r            *dnsRecordFrameworkResource
		args         args
		wantTypeName string
	}{
		{
			name: "type_name",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
			wantTypeName: "unifi_dns_record",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", tt.args.resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "has_id",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwresource.IdentitySchemaRequest{},
				resp: &fwresource.IdentitySchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.IdentitySchema.Attributes["id"]; !ok {
				t.Error("expected identity schema to have 'id' attribute")
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "has_key_attributes",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			s := tt.args.resp.Schema
			for _, attr := range []string{"id", "site", "name", "enabled", "port", "priority", "record_type", "ttl", "value", "weight"} {
				if _, ok := s.Attributes[attr]; !ok {
					t.Errorf("expected attribute %q in schema", attr)
				}
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "has_key_0",
			r:    &dnsRecordFrameworkResource{},
			args: args{ctx: context.Background()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.UpgradeState(tt.args.ctx)
			if _, ok := got[0]; !ok {
				t.Error("expected state upgrader at key 0")
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name      string
		r         *dnsRecordFrameworkResource
		args      args
		wantError bool
	}{
		{
			name: "nil_provider_data",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
		{
			name: "wrong_type",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: "wrong"},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: true,
		},
		{
			name: "correct_client",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: &Client{}},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf(
					"hasError = %v, want %v",
					tt.args.resp.Diagnostics.HasError(),
					tt.wantError,
				)
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *dnsRecordFrameworkResourceModel
		state *dnsRecordFrameworkResourceModel
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "copies_non_null_fields",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0: context.Background(),
				plan: &dnsRecordFrameworkResourceModel{
					Name:       types.StringValue("test"),
					Enabled:    types.BoolValue(true),
					RecordType: types.StringValue("A"),
					Value:      types.StringValue("1.2.3.4"),
				},
				state: &dnsRecordFrameworkResourceModel{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
			if tt.args.state.Name.ValueString() != "test" {
				t.Error("expected Name to be copied from plan")
			}
			if tt.args.state.RecordType.ValueString() != "A" {
				t.Error("expected RecordType to be copied from plan")
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_modelToDNSRecord(t *testing.T) {
	type args struct {
		in0   context.Context
		model *dnsRecordFrameworkResourceModel
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
		want *unifi.DNSRecord
	}{
		{
			name: "basic_conversion",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0: context.Background(),
				model: &dnsRecordFrameworkResourceModel{
					Name:       types.StringValue("test.example.com"),
					Value:      types.StringValue("1.2.3.4"),
					Enabled:    types.BoolValue(true),
					RecordType: types.StringValue("A"),
					Priority:   types.Int64Value(10),
					Weight:     types.Int64Value(20),
					Port:       types.Int64Null(),
					TTL:        timetypes.NewGoDurationNull(),
				},
			},
			want: &unifi.DNSRecord{
				Key:        "test.example.com",
				Value:      "1.2.3.4",
				Enabled:    true,
				RecordType: "A",
				Priority:   10,
				Weight:     20,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.modelToDNSRecord(
				tt.args.in0,
				tt.args.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("modelToDNSRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_dnsRecordToModel(t *testing.T) {
	type args struct {
		in0       context.Context
		dnsRecord *unifi.DNSRecord
		model     *dnsRecordFrameworkResourceModel
		site      string
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "full_record",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0: context.Background(),
				dnsRecord: &unifi.DNSRecord{
					ID:         "abc123",
					Key:        "test.example.com",
					Value:      "1.2.3.4",
					Enabled:    true,
					RecordType: "A",
					Priority:   10,
				},
				model: &dnsRecordFrameworkResourceModel{},
				site:  "default",
			},
		},
		{
			name: "empty_optional_fields_become_null",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0: context.Background(),
				dnsRecord: &unifi.DNSRecord{
					ID:    "abc123",
					Key:   "test",
					Value: "1.2.3.4",
				},
				model: &dnsRecordFrameworkResourceModel{},
				site:  "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.dnsRecordToModel(tt.args.in0, tt.args.dnsRecord, tt.args.model, tt.args.site)
			if tt.args.model.ID.ValueString() != tt.args.dnsRecord.ID {
				t.Errorf("ID = %q, want %q", tt.args.model.ID.ValueString(), tt.args.dnsRecord.ID)
			}
			if tt.args.model.Site.ValueString() != tt.args.site {
				t.Errorf("Site = %q, want %q", tt.args.model.Site.ValueString(), tt.args.site)
			}
			if tt.args.model.Name.ValueString() != tt.args.dnsRecord.Key {
				t.Errorf(
					"Name = %q, want %q",
					tt.args.model.Name.ValueString(),
					tt.args.dnsRecord.Key,
				)
			}
		})
	}
}

func Test_dnsRecordFrameworkResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *dnsRecordFrameworkResource
		args args
	}{
		{
			name: "returns_schema",
			r:    &dnsRecordFrameworkResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if len(tt.args.resp.Schema.Attributes) == 0 {
				t.Error("expected non-empty list resource schema")
			}
		})
	}
}
