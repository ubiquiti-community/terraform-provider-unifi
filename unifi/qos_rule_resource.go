package unifi

import (
	"context"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &qosRuleResource{}
	_ resource.ResourceWithImportState = &qosRuleResource{}
	_ resource.ResourceWithIdentity    = &qosRuleResource{}
)

func NewQOSRuleResource() resource.Resource {
	return &qosRuleResource{}
}

// qosRuleResource manages a QoS rule (Settings > Routing > QoS / Traffic
// Management): prioritize or rate-limit traffic from clients or networks to
// apps, app categories, IPs, regions or domains.
type qosRuleResource struct {
	client *Client
}

type qosRuleModel struct {
	ID                types.String   `tfsdk:"id"`
	Site              types.String   `tfsdk:"site"`
	Name              types.String   `tfsdk:"name"`
	Enabled           types.Bool     `tfsdk:"enabled"`
	Objective         types.String   `tfsdk:"objective"`
	DownloadLimitKbps types.Int64    `tfsdk:"download_limit_kbps"`
	UploadLimitKbps   types.Int64    `tfsdk:"upload_limit_kbps"`
	DownloadBurst     types.String   `tfsdk:"download_burst"`
	UploadBurst       types.String   `tfsdk:"upload_burst"`
	WANOrVPNNetwork   types.String   `tfsdk:"wan_or_vpn_network"`
	Index             types.Int64    `tfsdk:"index"`
	Source            types.Object   `tfsdk:"source"`
	Destination       types.Object   `tfsdk:"destination"`
	Schedule          types.Object   `tfsdk:"schedule"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
}

type qosRuleSourceModel struct {
	MatchingTarget types.String `tfsdk:"matching_target"`
	ClientMACs     types.Set    `tfsdk:"client_macs"`
	NetworkIDs     types.Set    `tfsdk:"network_ids"`
}

type qosRuleDestinationModel struct {
	MatchingTarget     types.String `tfsdk:"matching_target"`
	MatchingTargetType types.String `tfsdk:"matching_target_type"`
	AppIDs             types.Set    `tfsdk:"app_ids"`
	AppCategoryIDs     types.Set    `tfsdk:"app_category_ids"`
	IPs                types.Set    `tfsdk:"ips"`
	IPGroupID          types.String `tfsdk:"ip_group_id"`
	IID                types.String `tfsdk:"iid"`
	Regions            types.Set    `tfsdk:"regions"`
	WebDomains         types.Set    `tfsdk:"web_domains"`
	WebGroupID         types.String `tfsdk:"web_group_id"`
	PortMatchingType   types.String `tfsdk:"port_matching_type"`
	Port               types.String `tfsdk:"port"`
	PortGroupID        types.String `tfsdk:"port_group_id"`
}

type qosRuleIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

var (
	qosRuleSourceAttrTypes = map[string]attr.Type{
		"matching_target": types.StringType,
		"client_macs":     types.SetType{ElemType: types.StringType},
		"network_ids":     types.SetType{ElemType: types.StringType},
	}
	qosRuleDestinationAttrTypes = map[string]attr.Type{
		"matching_target":      types.StringType,
		"matching_target_type": types.StringType,
		"app_ids":              types.SetType{ElemType: types.Int64Type},
		"app_category_ids":     types.SetType{ElemType: types.Int64Type},
		"ips":                  types.SetType{ElemType: types.StringType},
		"ip_group_id":          types.StringType,
		"iid":                  types.StringType,
		"regions":              types.SetType{ElemType: types.StringType},
		"web_domains":          types.SetType{ElemType: types.StringType},
		"web_group_id":         types.StringType,
		"port_matching_type":   types.StringType,
		"port":                 types.StringType,
		"port_group_id":        types.StringType,
	}
)

func (r *qosRuleResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_qos_rule"
}

func (r *qosRuleResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id":   identityschema.StringAttribute{RequiredForImport: true},
			"site": identityschema.StringAttribute{OptionalForImport: true},
		},
	}
}

func (r *qosRuleResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_qos_rule` manages a QoS rule (Settings > Routing > QoS): prioritize " +
			"and/or rate-limit traffic from clients or networks to apps, app categories, IPs, regions or domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the QoS rule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the rule with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the rule.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the rule is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"objective": schema.StringAttribute{
				MarkdownDescription: "`PRIORITIZE`, `LIMIT` or `LIMIT_AND_PRIORITIZE`.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("PRIORITIZE", "LIMIT", "LIMIT_AND_PRIORITIZE"),
				},
			},
			"download_limit_kbps": schema.Int64Attribute{
				MarkdownDescription: "Download limit in kbps. Ignored by the controller for `PRIORITIZE`.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"upload_limit_kbps": schema.Int64Attribute{
				MarkdownDescription: "Upload limit in kbps. Ignored by the controller for `PRIORITIZE`.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"download_burst": schema.StringAttribute{
				MarkdownDescription: "Download burst allowance: `OFF`, `SHORT` or `LONG`. Defaults to `OFF`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("OFF"),
				Validators:          []validator.String{stringvalidator.OneOf("OFF", "SHORT", "LONG")},
			},
			"upload_burst": schema.StringAttribute{
				MarkdownDescription: "Upload burst allowance: `OFF`, `SHORT` or `LONG`. Defaults to `OFF`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("OFF"),
				Validators:          []validator.String{stringvalidator.OneOf("OFF", "SHORT", "LONG")},
			},
			"wan_or_vpn_network": schema.StringAttribute{
				MarkdownDescription: "ID of the WAN or VPN network the rule applies to. Omit for all interfaces.",
				Optional:            true,
			},
			"index": schema.Int64Attribute{
				MarkdownDescription: "Position of the rule, assigned by the controller.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "Traffic the rule applies to. Omit for all clients.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"matching_target": schema.StringAttribute{
						MarkdownDescription: "`ANY`, `CLIENT` (`client_macs`) or `NETWORK` (`network_ids`).",
						Required:            true,
						Validators:          []validator.String{stringvalidator.OneOf("ANY", "CLIENT", "NETWORK")},
					},
					"client_macs": schema.SetAttribute{
						MarkdownDescription: "Client MAC addresses, for `CLIENT`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"network_ids": schema.SetAttribute{
						MarkdownDescription: "Network IDs, for `NETWORK`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
				},
			},
			"destination": schema.SingleNestedAttribute{
				MarkdownDescription: "Where the traffic goes. Omit for any destination.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"matching_target": schema.StringAttribute{
						MarkdownDescription: "`ANY`, `APP` (`app_ids`), `APP_CATEGORY` (`app_category_ids`), " +
							"`IP` (`ips` or `ip_group_id`), `IID` (`iid`), `REGION` (`regions`) or " +
							"`WEB` (`web_domains` or `web_group_id`).",
						Required: true,
						Validators: []validator.String{stringvalidator.OneOf(
							"ANY", "APP", "APP_CATEGORY", "IP", "IID", "REGION", "WEB")},
					},
					"matching_target_type": schema.StringAttribute{
						MarkdownDescription: "For `IP`/`WEB`: `SPECIFIC` (listed values) or `OBJECT` (a group ID).",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.OneOf("SPECIFIC", "OBJECT")},
					},
					"app_ids": schema.SetAttribute{
						MarkdownDescription: "DPI application IDs, for `APP`.",
						ElementType:         types.Int64Type,
						Optional:            true,
					},
					"app_category_ids": schema.SetAttribute{
						MarkdownDescription: "DPI application category IDs, for `APP_CATEGORY`.",
						ElementType:         types.Int64Type,
						Optional:            true,
					},
					"ips": schema.SetAttribute{
						MarkdownDescription: "IP addresses or subnets, for `IP` with `SPECIFIC`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"ip_group_id": schema.StringAttribute{
						MarkdownDescription: "IP group ID, for `IP` with `OBJECT`.",
						Optional:            true,
					},
					"iid": schema.StringAttribute{
						MarkdownDescription: "IPv6 interface identifier, for `IID`.",
						Optional:            true,
					},
					"regions": schema.SetAttribute{
						MarkdownDescription: "Country codes, for `REGION`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"web_domains": schema.SetAttribute{
						MarkdownDescription: "Domains, for `WEB` with `SPECIFIC`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"web_group_id": schema.StringAttribute{
						MarkdownDescription: "Domain group ID, for `WEB` with `OBJECT`.",
						Optional:            true,
					},
					"port_matching_type": schema.StringAttribute{
						MarkdownDescription: "`SPECIFIC` (`port`) or `OBJECT` (`port_group_id`). Omit for any port.",
						Optional:            true,
						Validators:          []validator.String{stringvalidator.OneOf("SPECIFIC", "OBJECT")},
					},
					"port": schema.StringAttribute{
						MarkdownDescription: "Port or comma-separated ports, for `SPECIFIC`.",
						Optional:            true,
					},
					"port_group_id": schema.StringAttribute{
						MarkdownDescription: "Port group ID, for `OBJECT`.",
						Optional:            true,
					},
				},
			},
			"schedule": ruleScheduleSchemaAttribute(),
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
	}
}

func (r *qosRuleResource) Configure(
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
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *qosRuleResource) site(m *qosRuleModel) string {
	if s := m.Site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

func (r *qosRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan qosRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	site := r.site(&plan)
	rule := r.modelToAPI(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateQOSRule(ctx, site, rule)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating QoS Rule", err.Error())
		return
	}
	r.apiToModel(ctx, created, &plan, site, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, qosRuleIdentityModel{ID: plan.ID, Site: types.StringValue(site)})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *qosRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state qosRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var identity qosRuleIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
	}
	site := state.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}
	id := state.ID.ValueString()
	if id == "" {
		id = identity.ID.ValueString()
	}

	rule, err := r.client.GetQOSRule(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading QoS Rule", "Could not read QoS rule "+id+": "+err.Error())
		return
	}
	r.apiToModel(ctx, rule, &state, site, &resp.Diagnostics)
	if req.Identity == nil || req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, qosRuleIdentityModel{ID: state.ID, Site: types.StringValue(site)})...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *qosRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state qosRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	site := r.site(&state)
	rule := r.modelToAPI(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	rule.ID = state.ID.ValueString()
	// The controller orders rules by index; keep the current position.
	if v := state.Index.ValueInt64Pointer(); v != nil {
		rule.Index = v
	}
	updated, err := r.client.UpdateQOSRule(ctx, site, rule)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating QoS Rule", err.Error())
		return
	}
	r.apiToModel(ctx, updated, &plan, site, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *qosRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state qosRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, d := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := r.client.DeleteQOSRule(ctx, r.site(&state), state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError("Error Deleting QoS Rule", err.Error())
	}
}

func (r *qosRuleResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	id, site := req.ID, r.client.Site
	if id == "" {
		var identity qosRuleIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		id = identity.ID.ValueString()
		if s := identity.Site.ValueString(); s != "" {
			site = s
		}
	} else if parts := strings.Split(id, ":"); len(parts) == 2 {
		site, id = parts[0], parts[1]
	}
	if id == "" {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be 'id' or 'site:id'.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, qosRuleIdentityModel{
		ID: types.StringValue(id), Site: types.StringValue(site),
	})...)
}

func (r *qosRuleResource) modelToAPI(ctx context.Context, m *qosRuleModel, diags *diag.Diagnostics) *unifi.QOSRule {
	rule := &unifi.QOSRule{
		Name:              m.Name.ValueString(),
		Enabled:           m.Enabled.ValueBool(),
		Objective:         m.Objective.ValueString(),
		DownloadLimitKbps: m.DownloadLimitKbps.ValueInt64Pointer(),
		UploadLimitKbps:   m.UploadLimitKbps.ValueInt64Pointer(),
		DownloadBurst:     m.DownloadBurst.ValueString(),
		UploadBurst:       m.UploadBurst.ValueString(),
		WANOrVPNNetwork:   m.WANOrVPNNetwork.ValueString(),
	}

	src := &unifi.QOSRuleSource{MatchingTarget: "ANY", PortMatchingType: "ANY"}
	if isKnown(m.Source) {
		var s qosRuleSourceModel
		diags.Append(m.Source.As(ctx, &s, basetypes.ObjectAsOptions{})...)
		src.MatchingTarget = s.MatchingTarget.ValueString()
		src.ClientMACs = stringsFromSet(ctx, s.ClientMACs, diags)
		src.NetworkIDs = stringsFromSet(ctx, s.NetworkIDs, diags)
	}
	rule.Source = src

	dst := &unifi.QOSRuleDestination{MatchingTarget: "ANY", PortMatchingType: "ANY"}
	if isKnown(m.Destination) {
		var d qosRuleDestinationModel
		diags.Append(m.Destination.As(ctx, &d, basetypes.ObjectAsOptions{})...)
		dst = &unifi.QOSRuleDestination{
			MatchingTarget:     d.MatchingTarget.ValueString(),
			MatchingTargetType: d.MatchingTargetType.ValueString(),
			AppIDs:             int64sFromSet(ctx, d.AppIDs, diags),
			AppCategoryIDs:     int64sFromSet(ctx, d.AppCategoryIDs, diags),
			IPs:                stringsFromSet(ctx, d.IPs, diags),
			IPGroupID:          d.IPGroupID.ValueString(),
			Iid:                d.IID.ValueString(),
			Regions:            stringsFromSet(ctx, d.Regions, diags),
			WebDomains:         stringsFromSet(ctx, d.WebDomains, diags),
			WebGroupID:         d.WebGroupID.ValueString(),
			PortMatchingType:   d.PortMatchingType.ValueString(),
			Port:               d.Port.ValueString(),
			PortGroupID:        d.PortGroupID.ValueString(),
		}
		if dst.PortMatchingType == "" {
			dst.PortMatchingType = "ANY"
		}
	}
	rule.Destination = dst

	s := scheduleFromObject(ctx, m.Schedule, diags)
	rule.Schedule = &unifi.QOSRuleSchedule{
		Mode: s.Mode, RepeatOnDays: s.RepeatOnDays, TimeAllDay: s.TimeAllDay,
		TimeRangeStart: s.TimeRangeStart, TimeRangeEnd: s.TimeRangeEnd,
		Date: s.Date, DateStart: s.DateStart, DateEnd: s.DateEnd,
	}
	return rule
}

func (r *qosRuleResource) apiToModel(
	ctx context.Context,
	rule *unifi.QOSRule,
	m *qosRuleModel,
	site string,
	diags *diag.Diagnostics,
) {
	m.ID = types.StringValue(rule.ID)
	m.Site = types.StringValue(site)
	m.Name = types.StringValue(rule.Name)
	m.Enabled = types.BoolValue(rule.Enabled)
	m.Objective = types.StringValue(rule.Objective)
	m.DownloadLimitKbps = types.Int64PointerValue(rule.DownloadLimitKbps)
	m.UploadLimitKbps = types.Int64PointerValue(rule.UploadLimitKbps)
	m.DownloadBurst = types.StringValue(orDefault(rule.DownloadBurst, "OFF"))
	m.UploadBurst = types.StringValue(orDefault(rule.UploadBurst, "OFF"))
	m.WANOrVPNNetwork = stringOrNull(rule.WANOrVPNNetwork)
	m.Index = types.Int64PointerValue(rule.Index)

	src := qosRuleSourceModel{MatchingTarget: types.StringValue("ANY")}
	if rule.Source != nil {
		src = qosRuleSourceModel{
			MatchingTarget: types.StringValue(orDefault(rule.Source.MatchingTarget, "ANY")),
			ClientMACs:     setOrNull(ctx, rule.Source.ClientMACs, diags),
			NetworkIDs:     setOrNull(ctx, rule.Source.NetworkIDs, diags),
		}
	} else {
		src.ClientMACs, src.NetworkIDs = types.SetNull(types.StringType), types.SetNull(types.StringType)
	}
	obj, d := types.ObjectValueFrom(ctx, qosRuleSourceAttrTypes, src)
	diags.Append(d...)
	m.Source = obj

	dst := unifi.QOSRuleDestination{MatchingTarget: "ANY"}
	if rule.Destination != nil {
		dst = *rule.Destination
	}
	portType := types.StringNull()
	// ANY is the default; keep it null unless a port is matched, so a config
	// without port_matching_type does not drift.
	if dst.PortMatchingType != "" && dst.PortMatchingType != "ANY" {
		portType = types.StringValue(dst.PortMatchingType)
	}
	dm := qosRuleDestinationModel{
		MatchingTarget:     types.StringValue(orDefault(dst.MatchingTarget, "ANY")),
		MatchingTargetType: stringOrNull(dst.MatchingTargetType),
		AppIDs:             int64SetOrNull(ctx, dst.AppIDs, diags),
		AppCategoryIDs:     int64SetOrNull(ctx, dst.AppCategoryIDs, diags),
		IPs:                setOrNull(ctx, dst.IPs, diags),
		IPGroupID:          stringOrNull(dst.IPGroupID),
		IID:                stringOrNull(dst.Iid),
		Regions:            setOrNull(ctx, dst.Regions, diags),
		WebDomains:         setOrNull(ctx, dst.WebDomains, diags),
		WebGroupID:         stringOrNull(dst.WebGroupID),
		PortMatchingType:   portType,
		Port:               stringOrNull(dst.Port),
		PortGroupID:        stringOrNull(dst.PortGroupID),
	}
	obj, d = types.ObjectValueFrom(ctx, qosRuleDestinationAttrTypes, dm)
	diags.Append(d...)
	m.Destination = obj

	var s ruleSchedule
	if rule.Schedule != nil {
		s = ruleSchedule{
			Mode: rule.Schedule.Mode, RepeatOnDays: rule.Schedule.RepeatOnDays,
			TimeAllDay: rule.Schedule.TimeAllDay, TimeRangeStart: rule.Schedule.TimeRangeStart,
			TimeRangeEnd: rule.Schedule.TimeRangeEnd, Date: rule.Schedule.Date,
			DateStart: rule.Schedule.DateStart, DateEnd: rule.Schedule.DateEnd,
		}
	}
	m.Schedule = scheduleToObject(ctx, s, diags)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
