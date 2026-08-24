package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccDynamicDNS_dyndns(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDynamicDNSConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_dynamic_dns.test", "service", "dyndns"),
					resource.TestCheckResourceAttr(
						"unifi_dynamic_dns.test",
						"host_name",
						"test.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_dynamic_dns.test",
						"server",
						"dyndns.example.com",
					),
				),
			},
			// String-ID import (classic `terraform import` with the object ID).
			{
				ResourceName:      "unifi_dynamic_dns.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_dynamic_dns.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

const testAccDynamicDNSConfig = `
resource "unifi_dynamic_dns" "test" {
	service = "dyndns"
	
	host_name = "test.example.com"

	server   = "dyndns.example.com"
	login    = "testuser"
	password = "password"
}
`

func TestNewDynamicDNSResource(t *testing.T) {
	r := NewDynamicDNSResource()
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
}

func TestNewDynamicDNSListResource(t *testing.T) {
	r := NewDynamicDNSListResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure")
	}
}

func Test_dynamicDNSResource_Metadata(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "unifi"}, resp)
	if resp.TypeName != "unifi_dynamic_dns" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_dynamic_dns")
	}
}

func Test_dynamicDNSResource_Schema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	for _, attr := range []string{"id", "site", "interface", "service", "host_name", "server", "login", "password"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected attribute %q in schema", attr)
		}
	}
}

func Test_dynamicDNSResource_IdentitySchema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("expected identity schema to have 'id' attribute")
	}
	if _, ok := resp.IdentitySchema.Attributes["site"]; !ok {
		t.Error("expected identity schema to have 'site' attribute")
	}
}

func Test_dynamicDNSResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		req       fwresource.ConfigureRequest
		wantError bool
	}{
		{"nil_provider_data", fwresource.ConfigureRequest{}, false},
		{"wrong_type", fwresource.ConfigureRequest{ProviderData: "wrong"}, true},
		{"correct_client", fwresource.ConfigureRequest{ProviderData: &Client{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), tt.req, resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

// dynamicDNSImportHarness builds the empty state and null identity containers
// the framework hands to ImportState, so the import logic can be unit tested
// without a live controller.
func dynamicDNSImportHarness(t *testing.T) (tfsdk.State, tfsdk.ResourceIdentity) {
	t.Helper()
	ctx := context.Background()
	r := &dynamicDNSResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	var idResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &idResp)
	if idResp.Diagnostics.HasError() {
		t.Fatalf("identity schema: %v", idResp.Diagnostics)
	}

	state := tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}
	identity := tfsdk.ResourceIdentity{
		Schema: idResp.IdentitySchema,
		Raw:    tftypes.NewValue(idResp.IdentitySchema.Type().TerraformType(ctx), nil),
	}
	return state, identity
}

// dynamicDNSImportTestID is the object id used by the ImportState unit tests.
const dynamicDNSImportTestID = "0123456789abcdef01234567"

// dynamicDNSIdentityValue builds a populated identity container for
// identity-based import requests with id dynamicDNSImportTestID. Pass ""
// to leave site null.
func dynamicDNSIdentityValue(t *testing.T, site string) tfsdk.ResourceIdentity {
	t.Helper()
	ctx := context.Background()
	_, identity := dynamicDNSImportHarness(t)

	siteVal := tftypes.NewValue(tftypes.String, nil)
	if site != "" {
		siteVal = tftypes.NewValue(tftypes.String, site)
	}
	identity.Raw = tftypes.NewValue(
		identity.Schema.Type().TerraformType(ctx),
		map[string]tftypes.Value{
			"id":   tftypes.NewValue(tftypes.String, dynamicDNSImportTestID),
			"site": siteVal,
		},
	)
	return identity
}

func Test_dynamicDNSResource_ImportState(t *testing.T) {
	ctx := context.Background()
	const oid = "0123456789abcdef01234567"

	newRes := func() *dynamicDNSResource {
		return &dynamicDNSResource{client: &Client{Site: "default"}}
	}

	getString := func(t *testing.T, get func(context.Context, path.Path, any) diag.Diagnostics, p path.Path) types.String {
		t.Helper()
		var v types.String
		if d := get(ctx, p, &v); d.HasError() {
			t.Fatalf("get %s: %v", p, d)
		}
		return v
	}

	t.Run("identity import defaults omitted site to provider site", func(t *testing.T) {
		r := newRes()
		state, _ := dynamicDNSImportHarness(t)
		reqIdentity := dynamicDNSIdentityValue(t, "")
		respIdentity := dynamicDNSIdentityValue(t, "")
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "", Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("state site = %v, want default", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("identity site = %v, want default", got)
		}
	})

	t.Run("identity import honors explicit site", func(t *testing.T) {
		r := newRes()
		state, _ := dynamicDNSImportHarness(t)
		reqIdentity := dynamicDNSIdentityValue(t, "other")
		respIdentity := dynamicDNSIdentityValue(t, "other")
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "", Identity: &reqIdentity}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("state site = %v, want other", got)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("identity site = %v, want other", got)
		}
	})

	t.Run("string import by id mirrors identity", func(t *testing.T) {
		r := newRes()
		state, respIdentity := dynamicDNSImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("id"),
		); got.ValueString() != oid {
			t.Errorf("identity id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "default" {
			t.Errorf("identity site = %v, want default", got)
		}
	})

	t.Run("string import with site prefix", func(t *testing.T) {
		r := newRes()
		state, respIdentity := dynamicDNSImportHarness(t)
		resp := &fwresource.ImportStateResponse{State: state, Identity: &respIdentity}

		r.ImportState(ctx, fwresource.ImportStateRequest{ID: "other:" + oid}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diags: %v", resp.Diagnostics)
		}
		if got := getString(
			t,
			resp.State.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("state site = %v, want other", got)
		}
		if got := getString(t, resp.State.GetAttribute, path.Root("id")); got.ValueString() != oid {
			t.Errorf("state id = %v, want %s", got, oid)
		}
		if got := getString(
			t,
			resp.Identity.GetAttribute,
			path.Root("site"),
		); got.ValueString() != "other" {
			t.Errorf("identity site = %v, want other", got)
		}
	})
}

func Test_dynamicDNSResource_applyPlanToState(t *testing.T) {
	r := &dynamicDNSResource{}
	plan := &dynamicDNSResourceModel{
		Interface: types.StringValue("wan"),
		Service:   types.StringValue("dyndns"),
		HostName:  types.StringValue("test.example.com"),
		Server:    types.StringValue("dyndns.example.com"),
		Login:     types.StringValue("user"),
		Password:  types.StringValue("pass"),
	}
	state := &dynamicDNSResourceModel{}
	r.applyPlanToState(context.Background(), plan, state)
	if state.Service.ValueString() != "dyndns" {
		t.Error("expected Service to be copied from plan")
	}
	if state.HostName.ValueString() != "test.example.com" {
		t.Error("expected HostName to be copied from plan")
	}
}

func Test_dynamicDNSResource_modelToDynamicDNS(t *testing.T) {
	tests := []struct {
		name  string
		model *dynamicDNSResourceModel
		want  *unifi.DynamicDNS
	}{
		{
			name: "basic_conversion",
			model: &dynamicDNSResourceModel{
				ID:        types.StringValue("abc123"),
				Interface: types.StringValue("wan"),
				Service:   types.StringValue("dyndns"),
				HostName:  types.StringValue("test.example.com"),
				Server:    types.StringValue("dyndns.example.com"),
				Login:     types.StringValue("user"),
				Password:  types.StringValue("pass"),
			},
			want: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
				Server:    "dyndns.example.com",
				Login:     "user",
				Password:  "pass",
			},
		},
		{
			name: "null_optional_fields",
			model: &dynamicDNSResourceModel{
				ID:        types.StringValue("abc123"),
				Interface: types.StringValue("wan"),
				Service:   types.StringValue("dyndns"),
				HostName:  types.StringValue("test.example.com"),
				Server:    types.StringNull(),
				Login:     types.StringNull(),
				Password:  types.StringNull(),
			},
			want: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			if got := r.modelToDynamicDNS(
				context.Background(),
				tt.model,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("modelToDynamicDNS() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_dynamicDNSResource_dynamicDNSToModel(t *testing.T) {
	tests := []struct {
		name       string
		dynamicDNS *unifi.DynamicDNS
		site       string
	}{
		{
			name: "full_record",
			dynamicDNS: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
				Server:    "dyndns.example.com",
				Login:     "user",
				Password:  "pass",
			},
			site: "default",
		},
		{
			name: "empty_optional_fields",
			dynamicDNS: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
			},
			site: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			model := &dynamicDNSResourceModel{}
			r.dynamicDNSToModel(context.Background(), tt.dynamicDNS, model, tt.site)
			if model.ID.ValueString() != tt.dynamicDNS.ID {
				t.Errorf("ID = %q, want %q", model.ID.ValueString(), tt.dynamicDNS.ID)
			}
			if tt.site == "" && !model.Site.IsNull() {
				t.Error("expected Site to be null for empty site")
			}
			if tt.dynamicDNS.Server == "" && !model.Server.IsNull() {
				t.Error("expected Server to be null for empty server")
			}
		})
	}
}

func Test_dynamicDNSResource_ListResourceConfigSchema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if len(resp.Schema.Attributes) == 0 {
		t.Error("expected non-empty list resource schema")
	}
}

func TestAccDynamicDNSList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccDynamicDNSConfig,
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_dynamic_dns" "test" {
						provider = unifi
						config {
							filter {
								name  = "host_name"
								value = "test.example.com"
							}
						}
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_dynamic_dns.test", 1),
				},
			},
		},
	})
}
