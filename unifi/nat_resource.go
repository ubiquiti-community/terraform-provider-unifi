package unifi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &natResource{}
	_ resource.ResourceWithImportState = &natResource{}
	_ resource.ResourceWithIdentity    = &natResource{}
)

func NewNatResource() resource.Resource {
	return &natResource{}
}

type natResource struct {
	client *Client
}

type natIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

type natResourceModel struct {
	ID                    types.String   `tfsdk:"id"`
	Site                  types.String   `tfsdk:"site"`
	Description           types.String   `tfsdk:"description"`
	Enabled               types.Bool     `tfsdk:"enabled"`
	Type                  types.String   `tfsdk:"type"`
	IPVersion             types.String   `tfsdk:"ip_version"`
	Protocol              types.String   `tfsdk:"protocol"`
	InInterface           types.String   `tfsdk:"in_interface"`
	OutInterface          types.String   `tfsdk:"out_interface"`
	IPAddress             types.String   `tfsdk:"ip_address"`
	Port                  types.Int64    `tfsdk:"port"`
	Exclude               types.Bool     `tfsdk:"exclude"`
	Logging               types.Bool     `tfsdk:"logging"`
	PppoeUseBaseInterface types.Bool     `tfsdk:"pppoe_use_base_interface"`
	RuleIndex             types.Int64    `tfsdk:"rule_index"`
	SettingPreference     types.String   `tfsdk:"setting_preference"`
	SourceFilter          types.Object   `tfsdk:"source_filter"`
	DestinationFilter     types.Object   `tfsdk:"destination_filter"`
	Timeouts              timeouts.Value `tfsdk:"timeouts"`
}

// natFilterModel is the nested source_filter / destination_filter block. The API
// uses identical shapes for both.
type natFilterModel struct {
	FilterType       types.String `tfsdk:"filter_type"`
	Address          types.String `tfsdk:"address"`
	Port             types.Int64  `tfsdk:"port"`
	InvertAddress    types.Bool   `tfsdk:"invert_address"`
	InvertPort       types.Bool   `tfsdk:"invert_port"`
	NetworkConfID    types.String `tfsdk:"network_conf_id"`
	FirewallGroupIDs types.Set    `tfsdk:"firewall_group_ids"`
}

func natFilterAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"filter_type":        types.StringType,
		"address":            types.StringType,
		"port":               types.Int64Type,
		"invert_address":     types.BoolType,
		"invert_port":        types.BoolType,
		"network_conf_id":    types.StringType,
		"firewall_group_ids": types.SetType{ElemType: types.StringType},
	}
}

func natFilterSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"filter_type": schema.StringAttribute{
				MarkdownDescription: "How traffic is matched. Can be `NONE`, `ADDRESS_AND_PORT`, `FIREWALL_GROUPS` or `NETWORK_CONF`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"NONE",
						"ADDRESS_AND_PORT",
						"FIREWALL_GROUPS",
						"NETWORK_CONF",
					),
				},
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "The address or CIDR to match. Used with the `ADDRESS_AND_PORT` filter type.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The port to match. Used with the `ADDRESS_AND_PORT` filter type.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 65535)},
			},
			"invert_address": schema.BoolAttribute{
				MarkdownDescription: "Match every address except `address`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"invert_port": schema.BoolAttribute{
				MarkdownDescription: "Match every port except `port`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"network_conf_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the network to match. Used with the `NETWORK_CONF` filter type.",
				Optional:            true,
			},
			"firewall_group_ids": schema.SetAttribute{
				MarkdownDescription: "The IDs of the firewall groups to match. Used with the `FIREWALL_GROUPS` filter type.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *natResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_nat"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *natResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"id":   identityschema.StringAttribute{RequiredForImport: true},
			"site": identityschema.StringAttribute{OptionalForImport: true},
		},
	}
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity]. See
// siteIdentityUpgraders.
func (r *natResource) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return siteIdentityUpgraders(func() *Client { return r.client })
}

func (r *natResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a NAT rule (DNAT, SNAT or masquerade) on the gateway.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the NAT rule.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the NAT rule with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the NAT rule.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the NAT rule is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of NAT rule. Can be `DNAT`, `SNAT` or `MASQUERADE`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("DNAT", "SNAT", "MASQUERADE"),
				},
			},
			"ip_version": schema.StringAttribute{
				MarkdownDescription: "The IP version the rule applies to. Can be `IPV4` or `IPV6`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("IPV4", "IPV6"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "The protocol to match. Can be `all`, `tcp`, `udp` or `tcp_udp`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "tcp", "udp", "tcp_udp"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"in_interface": schema.StringAttribute{
				MarkdownDescription: "The ID of the network that incoming traffic arrives on. Used by `DNAT` rules.",
				Optional:            true,
			},
			"out_interface": schema.StringAttribute{
				MarkdownDescription: "The ID of the network that outgoing traffic leaves on. Used by `SNAT` and `MASQUERADE` rules.",
				Optional:            true,
			},
			"ip_address": schema.StringAttribute{
				MarkdownDescription: "The translated IP address.",
				Optional:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "The translated port.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.Between(1, 65535)},
			},
			"exclude": schema.BoolAttribute{
				MarkdownDescription: "Exclude matching traffic from translation. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"logging": schema.BoolAttribute{
				MarkdownDescription: "Whether to log matching traffic. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"pppoe_use_base_interface": schema.BoolAttribute{
				MarkdownDescription: "Use the base interface of a PPPoE WAN instead of the PPPoE interface. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"rule_index": schema.Int64Attribute{
				MarkdownDescription: "The position of the rule in the rule set. Assigned by the controller when not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"setting_preference": schema.StringAttribute{
				MarkdownDescription: "Whether the rule is managed automatically or manually. Can be `auto` or `manual`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "manual"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_filter":      natFilterSchema("Matches traffic by its source."),
			"destination_filter": natFilterSchema("Matches traffic by its destination."),
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

func (r *natResource) Configure(
	_ context.Context,
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

func (r *natResource) siteOrDefault(site types.String) string {
	if s := site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

func (r *natResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan natResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, d := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	nat, diags := natModelToAPI(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := r.siteOrDefault(plan.Site)
	created, err := r.client.CreateNat(ctx, site, nat)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating NAT Rule", err.Error())
		return
	}

	resp.Diagnostics.Append(natAPIToModel(ctx, created, &plan, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(
		resp.Identity.Set(ctx, natIdentityModel{ID: plan.ID, Site: plan.Site})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state natResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, d := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// A stored identity must be passed back unchanged: Terraform rejects any
	// modification of a non-null identity.
	haveIdentity := req.Identity != nil && !req.Identity.Raw.IsNull()
	var identity natIdentityModel
	if haveIdentity {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// State is identity-only right after an identity-based import.
	id := state.ID.ValueString()
	if id == "" {
		id = identity.ID.ValueString()
	}
	site := state.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}
	if id == "" {
		resp.Diagnostics.AddError("Invalid State", "NAT rule must have an ID")
		return
	}

	nat, err := r.client.GetNat(ctx, site, id)
	if err != nil {
		var notFound *unifi.NotFoundError
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NAT Rule",
			"Could not read NAT rule with ID "+id+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(natAPIToModel(ctx, nat, &state, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !haveIdentity {
		identity = natIdentityModel{ID: state.ID, Site: state.Site}
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *natResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state natResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, d := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	nat, diags := natModelToAPI(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	nat.ID = state.ID.ValueString()

	site := r.siteOrDefault(state.Site)
	updated, err := r.client.UpdateNat(ctx, site, nat)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating NAT Rule", err.Error())
		return
	}

	resp.Diagnostics.Append(natAPIToModel(ctx, updated, &plan, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Identity is immutable once set: carry the incoming one through.
	identity := natIdentityModel{ID: plan.ID, Site: plan.Site}
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *natResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state natResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, d := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err := r.client.DeleteNat(ctx, r.siteOrDefault(state.Site), state.ID.ValueString())
	if err != nil {
		var notFound *unifi.NotFoundError
		if errors.As(err, &notFound) {
			return
		}
		resp.Diagnostics.AddError("Error Deleting NAT Rule", err.Error())
	}
}

func (r *natResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Import by ID string: "site:id" or just "id" for the default site.
	if req.ID != "" {
		var identity natIdentityModel
		switch parts := strings.Split(req.ID, ":"); len(parts) {
		case 1:
			identity.ID = types.StringValue(parts[0])
		case 2:
			identity.Site = types.StringValue(parts[0])
			identity.ID = types.StringValue(parts[1])
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...)
		default:
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				"Import ID must be in format 'site:id' or 'id'",
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
		return
	}

	// Import by resource identity (import block with identity, Terraform 1.12+).
	var identity natIdentityModel
	resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
	if site := identity.Site.ValueString(); site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}
}

// natModelToAPI converts the Terraform model to the API struct. Unset ip_version,
// protocol, setting_preference and filters are sent as the UI's defaults: a rule
// without them was answered with an unhandled HTTP 500.
func natModelToAPI(ctx context.Context, model *natResourceModel) (*unifi.Nat, diag.Diagnostics) {
	var diags diag.Diagnostics

	nat := &unifi.Nat{
		Description:           model.Description.ValueString(),
		Enabled:               model.Enabled.ValueBool(),
		Type:                  model.Type.ValueString(),
		Version:               natStringOrDefault(model.IPVersion, "IPV4"),
		Protocol:              natStringOrDefault(model.Protocol, "all"),
		InInterface:           model.InInterface.ValueString(),
		OutInterface:          model.OutInterface.ValueString(),
		IPAddress:             model.IPAddress.ValueString(),
		Exclude:               model.Exclude.ValueBool(),
		Logging:               model.Logging.ValueBool(),
		PppoeUseBaseInterface: model.PppoeUseBaseInterface.ValueBool(),
		SettingPreference:     natStringOrDefault(model.SettingPreference, "manual"),
	}

	if !model.Port.IsNull() && !model.Port.IsUnknown() {
		nat.Port = model.Port.ValueInt64Pointer()
	}
	if !model.RuleIndex.IsNull() && !model.RuleIndex.IsUnknown() {
		nat.RuleIndex = model.RuleIndex.ValueInt64Pointer()
	}

	src, d := natFilterToAPI(ctx, model.SourceFilter)
	diags.Append(d...)
	nat.SourceFilter = (*unifi.NatSourceFilter)(src)

	dst, d := natFilterToAPI(ctx, model.DestinationFilter)
	diags.Append(d...)
	nat.DestinationFilter = (*unifi.NatDestinationFilter)(dst)

	return nat, diags
}

// natStringOrDefault returns the value, or def when it is null or unknown.
func natStringOrDefault(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return def
	}
	return v.ValueString()
}

// natFilterToAPI returns the unfiltered `NONE` filter when none is configured.
func natFilterToAPI(
	ctx context.Context,
	obj types.Object,
) (*unifi.NatSourceFilter, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return &unifi.NatSourceFilter{FilterType: "NONE"}, diags
	}

	var m natFilterModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	f := &unifi.NatSourceFilter{
		FilterType:    m.FilterType.ValueString(),
		Address:       m.Address.ValueString(),
		InvertAddress: m.InvertAddress.ValueBool(),
		InvertPort:    m.InvertPort.ValueBool(),
		NetworkConfID: m.NetworkConfID.ValueString(),
	}
	if !m.Port.IsNull() && !m.Port.IsUnknown() {
		f.Port = m.Port.ValueInt64Pointer()
	}
	if !m.FirewallGroupIDs.IsNull() && !m.FirewallGroupIDs.IsUnknown() {
		diags.Append(m.FirewallGroupIDs.ElementsAs(ctx, &f.FirewallGroupIDs, false)...)
	}

	return f, diags
}

// natAPIToModel copies the API struct into the model. Empty values become null so
// that state matches what an unset configuration produces.
func natAPIToModel(
	ctx context.Context,
	nat *unifi.Nat,
	model *natResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(nat.ID)
	model.Site = types.StringValue(site)
	model.Description = types.StringValue(nat.Description)
	model.Enabled = types.BoolValue(nat.Enabled)
	model.Type = types.StringValue(nat.Type)
	model.IPVersion = stringOrNull(nat.Version)
	model.Protocol = stringOrNull(nat.Protocol)
	model.InInterface = stringOrNull(nat.InInterface)
	model.OutInterface = stringOrNull(nat.OutInterface)
	model.IPAddress = stringOrNull(nat.IPAddress)
	model.Port = types.Int64PointerValue(nat.Port)
	model.Exclude = types.BoolValue(nat.Exclude)
	model.Logging = types.BoolValue(nat.Logging)
	model.PppoeUseBaseInterface = types.BoolValue(nat.PppoeUseBaseInterface)
	model.RuleIndex = types.Int64PointerValue(nat.RuleIndex)
	model.SettingPreference = stringOrNull(nat.SettingPreference)

	var d diag.Diagnostics
	model.SourceFilter, d = natFilterFromAPI(ctx, (*unifi.NatSourceFilter)(nat.SourceFilter),
		model.SourceFilter)
	diags.Append(d...)
	model.DestinationFilter, d = natFilterFromAPI(
		ctx,
		(*unifi.NatSourceFilter)(nat.DestinationFilter),
		model.DestinationFilter,
	)
	diags.Append(d...)

	return diags
}

// natFilterFromAPI keeps the filter null when it was not configured and the
// controller only reports the empty `NONE` default, which avoids a spurious diff.
func natFilterFromAPI(
	ctx context.Context,
	f *unifi.NatSourceFilter,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	null := types.ObjectNull(natFilterAttrTypes())

	if f == nil {
		return null, nil
	}
	if (prior.IsNull() || prior.IsUnknown()) && (f.FilterType == "" || f.FilterType == "NONE") {
		return null, nil
	}

	groups := types.SetNull(types.StringType)
	var diags diag.Diagnostics
	if len(f.FirewallGroupIDs) > 0 {
		groups, diags = types.SetValueFrom(ctx, types.StringType, f.FirewallGroupIDs)
		if diags.HasError() {
			return null, diags
		}
	}

	obj, d := types.ObjectValueFrom(ctx, natFilterAttrTypes(), natFilterModel{
		FilterType:       types.StringValue(f.FilterType),
		Address:          stringOrNull(f.Address),
		Port:             types.Int64PointerValue(f.Port),
		InvertAddress:    types.BoolValue(f.InvertAddress),
		InvertPort:       types.BoolValue(f.InvertPort),
		NetworkConfID:    stringOrNull(f.NetworkConfID),
		FirewallGroupIDs: groups,
	})
	diags.Append(d...)

	return obj, diags
}
