package unifi

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// frrConfigTemplate is the Go template used to render FRR config from structured attributes.
var frrConfigTemplate = template.Must(template.New("frr").Parse(strings.TrimSpace(`
frr defaults traditional
log file stdout
!
router bgp {{.ASN}}
  bgp ebgp-requires-policy
  bgp router-id {{.RouterID}}
  bgp log-neighbor-changes
  bgp graceful-restart
  bgp bestpath as-path multipath-relax
{{- range .Neighbors}}
  !
  neighbor {{.Name}} peer-group
  neighbor {{.Name}} remote-as {{.RemoteAS}}
  neighbor {{.Name}} ebgp-multihop 2
  neighbor {{.Name}} timers 3 9
  neighbor {{.Name}} timers connect 5
  neighbor {{.Name}} soft-reconfiguration inbound
  {{- if .Description}}
  neighbor {{.Name}} description {{.Description}}
  {{- end}}
  !
  {{- $peer := .Name }}
  {{- range .Networks}}
  bgp listen range {{.}} peer-group {{$peer}}
  {{- end}}
  !
  address-family ipv4 unicast
    redistribute connected
    neighbor {{.Name}} activate
    neighbor {{.Name}} route-map {{.Name}}-IN in
    neighbor {{.Name}} route-map {{.Name}}-OUT out
    neighbor {{.Name}} maximum-prefix 1000
    neighbor {{.Name}} next-hop-self
  exit-address-family
  !
  address-family ipv6 unicast
    redistribute connected
    neighbor {{.Name}} activate
    neighbor {{.Name}} route-map {{.Name}}-IN-V6 in
    neighbor {{.Name}} route-map {{.Name}}-OUT-V6 out
    neighbor {{.Name}} maximum-prefix 1000
    neighbor {{.Name}} next-hop-self
  exit-address-family
!
route-map {{.Name}}-IN permit 10
!
route-map {{.Name}}-OUT permit 10
!
route-map {{.Name}}-IN-V6 permit 10
!
route-map {{.Name}}-OUT-V6 permit 10
{{- end}}
!
line vty
!
`)))

// frrTemplateData is the data structure passed to the FRR config template.
type frrTemplateData struct {
	ASN       int64
	RouterID  string
	Neighbors []frrNeighborData
}

type frrNeighborData struct {
	Name        string
	RemoteAS    int64
	Description string
	Networks    []string
}

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &bgpResource{}
	_ resource.ResourceWithImportState = &bgpResource{}
)

func NewBGPResource() resource.Resource {
	return &bgpResource{}
}

// bgpResource defines the resource implementation.
type bgpResource struct {
	client *Client
}

// bgpPeerModel describes a single BGP peer in the peers list.
type bgpPeerModel struct {
	Name        types.String `tfsdk:"name"`
	RemoteAS    types.Int64  `tfsdk:"remote_as"`
	Description types.String `tfsdk:"description"`
	Networks    types.List   `tfsdk:"networks"`
}

func (m bgpPeerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"remote_as":   types.Int64Type,
		"description": types.StringType,
		"networks":    types.ListType{ElemType: types.StringType},
	}
}

// bgpResourceModel describes the resource data model.
type bgpResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Config         types.String `tfsdk:"config"`
	ASN            types.Int64  `tfsdk:"asn"`
	RouterID       types.String `tfsdk:"router_id"`
	Peers          types.List   `tfsdk:"peers"`
	UploadFileName types.String `tfsdk:"upload_file_name"`
	Description    types.String `tfsdk:"description"`
}

func (r *bgpResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_bgp"
}

func (r *bgpResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages BGP configuration for the UniFi Controller. " +
			"Configuration can be provided either as a raw FRR config string via `config`, " +
			"or via structured attributes (`asn`, `router_id`, `peers`) which render a config from a template.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the BGP configuration.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the BGP configuration with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Enable BGP routing.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "The raw FRRouting BGP daemon configuration. Conflicts with `asn`, `router_id`, and `peers`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("asn"),
						path.MatchRoot("router_id"),
						path.MatchRoot("peers"),
					),
				},
			},
			"asn": schema.Int64Attribute{
				MarkdownDescription: "The BGP Autonomous System Number. Conflicts with `config`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 4294967295),
					int64validator.AlsoRequires(
						path.MatchRoot("router_id"),
						path.MatchRoot("peers"),
					),
				},
			},
			"router_id": schema.StringAttribute{
				MarkdownDescription: "The BGP router ID (typically an IP address). Conflicts with `config`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.AlsoRequires(
						path.MatchRoot("asn"),
						path.MatchRoot("peers"),
					),
				},
			},
			"peers": schema.ListNestedAttribute{
				MarkdownDescription: "List of BGP peer groups. Conflicts with `config`.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The peer group name.",
							Required:            true,
						},
						"remote_as": schema.Int64Attribute{
							MarkdownDescription: "The remote Autonomous System Number for this peer group.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 4294967295),
							},
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of this peer group.",
							Optional:            true,
						},
						"networks": schema.ListAttribute{
							MarkdownDescription: "List of network CIDR ranges to listen on for this peer group.",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"upload_file_name": schema.StringAttribute{
				MarkdownDescription: "The name of the uploaded configuration file.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("frr.conf"),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the BGP configuration.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("BGP Configuration"),
			},
		},
	}
}

func (r *bgpResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client
}

func (r *bgpResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data bgpResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert to unifi.BGPConfig
	bgpConfig, d := r.modelToBGP(ctx, &data)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Create the BGP configuration
	// Create the BGP configuration with retry for "not found" errors
	var createdBGPConfig *unifi.BGPConfig
	var err error

	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		createdBGPConfig, err = r.client.CreateBGPConfig(ctx, site, bgpConfig)
		if err == nil {
			break
		}

		// Retry only on "not found" errors
		if _, ok := err.(*unifi.NotFoundError); ok && attempt < maxRetries {
			continue
		}

		resp.Diagnostics.AddError(
			"Error Creating BGP Configuration",
			err.Error(),
		)
		return
	}

	// Convert back to model
	r.bgpToModel(ctx, createdBGPConfig, &data, site)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bgpResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data bgpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Get the BGP configuration from the API
	bgpConfig, err := r.client.GetBGPConfig(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading BGP Configuration",
			"Could not read BGP configuration: "+err.Error(),
		)
		return
	}

	// Convert to model
	r.bgpToModel(ctx, bgpConfig, &data, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *bgpResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state bgpResourceModel
	var plan bgpResourceModel

	// Read the current state
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply the plan changes to the state object
	r.applyPlanToState(ctx, &plan, &state)

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert the updated state to API format
	bgpConfig, d := r.modelToBGP(ctx, &state)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	bgpConfig.ID = state.ID.ValueString()

	// Send to API
	updatedBGPConfig, err := r.client.UpdateBGPConfig(ctx, site, bgpConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating BGP Configuration",
			err.Error(),
		)
		return
	}

	// Update state with API response
	r.bgpToModel(ctx, updatedBGPConfig, &state, site)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bgpResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data bgpResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Delete the BGP configuration
	err := r.client.DeleteBGPConfig(ctx, site)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting BGP Configuration",
			err.Error(),
		)
		return
	}
}

func (r *bgpResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(
		ctx,
		path.Root("id"),
		req,
		resp,
	)
}

// applyPlanToState merges plan values into state, preserving state values where plan is null/unknown.
func (r *bgpResource) applyPlanToState(
	_ context.Context,
	plan *bgpResourceModel,
	state *bgpResourceModel,
) {
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		state.Enabled = plan.Enabled
	}
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		state.Config = plan.Config
	}
	if !plan.ASN.IsNull() && !plan.ASN.IsUnknown() {
		state.ASN = plan.ASN
	}
	if !plan.RouterID.IsNull() && !plan.RouterID.IsUnknown() {
		state.RouterID = plan.RouterID
	}
	if !plan.Peers.IsNull() && !plan.Peers.IsUnknown() {
		state.Peers = plan.Peers
	}
	if !plan.UploadFileName.IsNull() && !plan.UploadFileName.IsUnknown() {
		state.UploadFileName = plan.UploadFileName
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		state.Description = plan.Description
	}
}

// renderFRRConfig renders the FRR config from the structured attributes.
func (r *bgpResource) renderFRRConfig(
	ctx context.Context,
	model *bgpResourceModel,
) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	var peers []bgpPeerModel
	d := model.Peers.ElementsAs(ctx, &peers, false)
	diags.Append(d...)
	if diags.HasError() {
		return "", diags
	}

	data := frrTemplateData{
		ASN:      model.ASN.ValueInt64(),
		RouterID: model.RouterID.ValueString(),
	}

	for _, peer := range peers {
		nd := frrNeighborData{
			Name:        peer.Name.ValueString(),
			RemoteAS:    peer.RemoteAS.ValueInt64(),
			Description: peer.Description.ValueString(),
		}

		if !peer.Networks.IsNull() && !peer.Networks.IsUnknown() {
			d := peer.Networks.ElementsAs(ctx, &nd.Networks, false)
			diags.Append(d...)
			if diags.HasError() {
				return "", diags
			}
		}

		data.Neighbors = append(data.Neighbors, nd)
	}

	var buf bytes.Buffer
	if err := frrConfigTemplate.Execute(&buf, data); err != nil {
		diags.AddError("Error Rendering FRR Config", err.Error())
		return "", diags
	}

	return buf.String(), diags
}

// modelToBGP converts the Terraform model to the API struct.
func (r *bgpResource) modelToBGP(
	ctx context.Context,
	model *bgpResourceModel,
) (*unifi.BGPConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	configStr := model.Config.ValueString()

	// If structured attributes are set, render the template.
	if !model.ASN.IsNull() && !model.ASN.IsUnknown() {
		rendered, d := r.renderFRRConfig(ctx, model)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		configStr = rendered
	}

	bgpConfig := &unifi.BGPConfig{
		Enabled:          model.Enabled.ValueBool(),
		Config:           configStr,
		UploadedFileName: model.UploadFileName.ValueString(),
		Description:      model.Description.ValueString(),
	}

	return bgpConfig, diags
}

// bgpToModel converts the API struct to the Terraform model.
// Structured attributes (asn, router_id, peers) are preserved from current state
// since the API only stores the rendered config string.
func (r *bgpResource) bgpToModel(
	_ context.Context,
	bgpConfig *unifi.BGPConfig,
	model *bgpResourceModel,
	site string,
) {
	model.ID = types.StringValue(bgpConfig.ID)
	model.Site = types.StringValue(site)
	model.Enabled = types.BoolValue(bgpConfig.Enabled)

	if bgpConfig.Config != "" {
		model.Config = types.StringValue(bgpConfig.Config)
	} else {
		model.Config = types.StringNull()
	}

	// ASN, RouterID, and Peers are preserved from state — the API only stores
	// the rendered config, so we don't attempt to parse it back.

	if bgpConfig.UploadedFileName != "" {
		model.UploadFileName = types.StringValue(bgpConfig.UploadedFileName)
	} else {
		model.UploadFileName = types.StringNull()
	}

	if bgpConfig.Description != "" {
		model.Description = types.StringValue(bgpConfig.Description)
	} else {
		model.Description = types.StringNull()
	}
}
