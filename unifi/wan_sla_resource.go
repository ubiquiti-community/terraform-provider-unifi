package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var (
	_ resource.Resource                   = &wanSlaResource{}
	_ resource.ResourceWithImportState    = &wanSlaResource{}
	_ resource.ResourceWithIdentity       = &wanSlaResource{}
	_ resource.ResourceWithValidateConfig = &wanSlaResource{}
)

func NewWANSlaResource() resource.Resource {
	return &wanSlaResource{}
}

type wanSlaResource struct {
	client *Client
}

type wanSlaResourceModel struct {
	ID                   types.String         `tfsdk:"id"`
	Site                 types.String         `tfsdk:"site"`
	Name                 types.String         `tfsdk:"name"`
	MonitorHealthType    types.String         `tfsdk:"monitor_health_type"`
	MonitorDownThreshold types.Int64          `tfsdk:"monitor_down_threshold"`
	Monitors             []wanSlaMonitorModel `tfsdk:"monitors"`
	Timeouts             timeouts.Value       `tfsdk:"timeouts"`
}

type wanSlaMonitorModel struct {
	MonitorID        types.Int64  `tfsdk:"monitor_id"`
	Type             types.String `tfsdk:"type"`
	Target           types.String `tfsdk:"target"`
	Domain           types.String `tfsdk:"domain"`
	Interval         types.Int64  `tfsdk:"interval"`
	TimePeriod       types.Int64  `tfsdk:"time_period"`
	ThresholdPolicy  types.String `tfsdk:"threshold_policy"`
	LossThreshold    types.Int64  `tfsdk:"loss_threshold"`
	LatencyThreshold types.Int64  `tfsdk:"latency_threshold"`
	JitterThreshold  types.Int64  `tfsdk:"jitter_threshold"`
}

type wanSlaIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

var wanSlaThresholdPolicies = []string{
	"any", "all", "any-two",
	"loss-or-all-others", "latency-or-all-others", "jitter-or-all-others",
	"loss-plus-one", "latency-plus-one", "jitter-plus-one",
}

func (r *wanSlaResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_wan_sla"
}

func (r *wanSlaResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Version: 2,
		Attributes: map[string]identityschema.Attribute{
			"id":   identityschema.StringAttribute{RequiredForImport: true},
			"site": identityschema.StringAttribute{OptionalForImport: true},
		},
	}
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity]. See siteIdentityUpgraders.
func (r *wanSlaResource) UpgradeIdentity(_ context.Context) map[int64]resource.IdentityUpgrader {
	return siteIdentityUpgraders(func() *Client { return r.client })
}

func (r *wanSlaResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_wan_sla` manages a WAN SLA profile (Network 11: Settings > Profiles > WAN SLA). " +
			"A profile has one or more verification servers, ICMP or DNS, each with packet loss, latency and jitter " +
			"thresholds. When enough servers breach their thresholds, the WANs using the profile (`unifi_wan.sla_id`) " +
			"are considered unusable and taken out of load balancing or failover.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the WAN SLA profile.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the profile belongs to.",
				Computed:            true,
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the profile.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"monitor_health_type": schema.StringAttribute{
				MarkdownDescription: "When the profile is considered down: `ALL` (every server breaches), `ANY` (any one), " +
					"`MAJORITY` (more than half) or `CUSTOM` (`monitor_down_threshold` servers).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("ALL", "ANY", "MAJORITY", "CUSTOM"),
				},
			},
			"monitor_down_threshold": schema.Int64Attribute{
				MarkdownDescription: "With `monitor_health_type = \"CUSTOM\"`: how many servers must breach for the profile to be down.",
				Optional:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"monitors": schema.ListNestedAttribute{
				MarkdownDescription: "The verification servers.",
				Required:            true,
				Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"monitor_id": schema.Int64Attribute{
							MarkdownDescription: "The controller's ID for this server.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "`ICMP` (ping) or `DNS` (a query for `domain`).",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("ICMP", "DNS"),
							},
						},
						"target": schema.StringAttribute{
							MarkdownDescription: "ICMP: the host name or IPv4 address pinged. DNS: the IPv4 address of the DNS server queried.",
							Required:            true,
						},
						"domain": schema.StringAttribute{
							MarkdownDescription: "DNS only: the name resolved against `target`.",
							Optional:            true,
						},
						"interval": schema.Int64Attribute{
							MarkdownDescription: "Seconds between probes (1-3600).",
							Required:            true,
							Validators:          []validator.Int64{int64validator.Between(1, 3600)},
						},
						"time_period": schema.Int64Attribute{
							MarkdownDescription: "Seconds over which probe results are averaged (1-65535). The controller requires it to exceed twice the interval plus the latency threshold in seconds.",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"threshold_policy": schema.StringAttribute{
							MarkdownDescription: "Which of the thresholds set must be exceeded for this server to breach: `any`, `all`, " +
								"`any-two`, `loss-or-all-others`, `latency-or-all-others`, `jitter-or-all-others`, `loss-plus-one`, " +
								"`latency-plus-one` or `jitter-plus-one` (e.g. `loss-or-all-others` = packet loss, or both latency and jitter).",
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf(wanSlaThresholdPolicies...),
							},
						},
						"loss_threshold": schema.Int64Attribute{
							MarkdownDescription: "Packet loss threshold in percent (1-100). Unset disables it.",
							Optional:            true,
							Validators:          []validator.Int64{int64validator.Between(1, 100)},
						},
						"latency_threshold": schema.Int64Attribute{
							MarkdownDescription: "Latency threshold in milliseconds (5-10000). Unset disables it.",
							Optional:            true,
							Validators: []validator.Int64{
								int64validator.Between(5, 10000),
							},
						},
						"jitter_threshold": schema.Int64Attribute{
							MarkdownDescription: "Jitter threshold in milliseconds (3-1500). Unset disables it.",
							Optional:            true,
							Validators:          []validator.Int64{int64validator.Between(3, 1500)},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(
				ctx,
				timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
			),
		},
	}
}

// ValidateConfig checks what the controller's form checks, so mistakes fail at plan instead of apply.
func (r *wanSlaResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var cfg wanSlaResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cfg.MonitorHealthType.ValueString() == "CUSTOM" && cfg.MonitorDownThreshold.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_down_threshold"),
			"Missing monitor_down_threshold",
			"monitor_health_type = \"CUSTOM\" needs monitor_down_threshold.",
		)
	}
	if !cfg.MonitorHealthType.IsUnknown() && cfg.MonitorHealthType.ValueString() != "CUSTOM" &&
		!cfg.MonitorDownThreshold.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("monitor_down_threshold"),
			"Unexpected monitor_down_threshold",
			"monitor_down_threshold only applies with monitor_health_type = \"CUSTOM\".",
		)
	}
	for i, m := range cfg.Monitors {
		p := path.Root("monitors").AtListIndex(i)
		switch m.Type.ValueString() {
		case "DNS":
			if m.Domain.IsNull() {
				resp.Diagnostics.AddAttributeError(
					p.AtName("domain"),
					"Missing domain",
					"A DNS server needs the domain it resolves.",
				)
			}
		case "ICMP":
			if !m.Domain.IsNull() {
				resp.Diagnostics.AddAttributeError(
					p.AtName("domain"),
					"Unexpected domain",
					"domain only applies to DNS servers.",
				)
			}
		}
		if m.LossThreshold.IsNull() && m.LatencyThreshold.IsNull() && m.JitterThreshold.IsNull() {
			resp.Diagnostics.AddAttributeError(
				p,
				"No threshold",
				"Set at least one of loss_threshold, latency_threshold, jitter_threshold.",
			)
		}
	}
}

func (r *wanSlaResource) Configure(
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

func (r *wanSlaResource) site(m *wanSlaResourceModel) string {
	if s := m.Site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

func (r *wanSlaResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan wanSlaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, d := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	site := r.site(&plan)
	created, err := r.client.CreateWANSla(ctx, site, wanSlaModelToAPI(&plan, nil))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating WAN SLA",
			fmt.Sprintf("Could not create WAN SLA profile: %s", err),
		)
		return
	}
	plan.Site = types.StringValue(site)
	wanSlaAPIToModel(created, &plan)
	resp.Diagnostics.Append(
		resp.Identity.Set(ctx, wanSlaIdentityModel{ID: plan.ID, Site: plan.Site})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *wanSlaResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state wanSlaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, d := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	if state.ID.IsNull() || state.ID.IsUnknown() {
		if req.Identity != nil && !req.Identity.Raw.IsFullyNull() {
			var identity wanSlaIdentityModel
			resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.ID = identity.ID
			if (state.Site.IsNull() || state.Site.IsUnknown()) &&
				identity.Site.ValueString() != "" {
				state.Site = identity.Site
			}
		}
	}

	site := r.site(&state)
	sla, err := r.client.GetWANSla(ctx, site, state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			removeResourceOnRead(ctx, resp, wanSlaIdentityModel{ID: state.ID, Site: state.Site})
			return
		}
		resp.Diagnostics.AddError("Error Reading WAN SLA",
			fmt.Sprintf("Could not read WAN SLA profile %s: %s", state.ID.ValueString(), err))
		return
	}
	state.Site = types.StringValue(site)
	wanSlaAPIToModel(sla, &state)
	if req.Identity == nil || req.Identity.Raw.IsFullyNull() {
		resp.Diagnostics.Append(
			resp.Identity.Set(ctx, wanSlaIdentityModel{ID: state.ID, Site: state.Site})...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *wanSlaResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state wanSlaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, d := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	site := r.site(&state)
	id := state.ID.ValueString()
	// Keep each server's controller id, matched by type and target rather than position, so reordering the list
	// doesn't swap ids. The controller's copy is authoritative.
	current, err := r.client.GetWANSla(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading WAN SLA",
			fmt.Sprintf("Could not read WAN SLA profile %s: %s", id, err),
		)
		return
	}
	sla := wanSlaModelToAPI(&plan, current.Monitors)
	sla.ID = id
	updated, err := r.client.UpdateWANSla(ctx, site, sla)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating WAN SLA",
			fmt.Sprintf("Could not update WAN SLA profile %s: %s", id, err),
		)
		return
	}
	plan.ID = state.ID
	plan.Site = types.StringValue(site)
	wanSlaAPIToModel(updated, &plan)
	if req.Identity == nil || req.Identity.Raw.IsFullyNull() {
		resp.Diagnostics.Append(
			resp.Identity.Set(ctx, wanSlaIdentityModel{ID: plan.ID, Site: plan.Site})...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *wanSlaResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state wanSlaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, d := state.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	err := r.client.DeleteWANSla(ctx, r.site(&state), state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting WAN SLA",
			fmt.Sprintf(
				"Could not delete WAN SLA profile %s (a WAN may still use it: detach it first): %s",
				state.ID.ValueString(),
				err,
			),
		)
	}
}

func (r *wanSlaResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	if req.ID == "" {
		var identity wanSlaIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		if identity.Site.ValueString() != "" {
			resp.Diagnostics.Append(
				resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...)
		}
		return
	}
	// "id" or "site:id".
	idParts, diags := util.ParseImportID(req.ID, 1, 2)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if site := idParts["site"]; site != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("site"), site)...)
	}
	if id := idParts["id"]; id != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
		resp.Diagnostics.Append(resp.Identity.SetAttribute(ctx, path.Root("id"), id)...)
	}
}

func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

// wanSlaModelToAPI builds the request. current, when given, is the controller's copy: each server keeps its monitor_id
// when one with the same type and target exists there.
func wanSlaModelToAPI(m *wanSlaResourceModel, current []unifi.WANSlaMonitor) *unifi.WANSla {
	ids := map[string]*int64{}
	for _, c := range current {
		ids[c.Type+"|"+c.Target] = c.MonitorID
	}
	out := &unifi.WANSla{
		Name:              m.Name.ValueString(),
		MonitorHealthType: m.MonitorHealthType.ValueString(),
		Monitors:          make([]unifi.WANSlaMonitor, 0, len(m.Monitors)),
	}
	if out.MonitorHealthType == unifi.WANSlaMonitorHealthCustom {
		out.MonitorDownThreshold = int64Ptr(m.MonitorDownThreshold)
	}
	for _, mon := range m.Monitors {
		am := unifi.WANSlaMonitor{
			MonitorID:  ids[mon.Type.ValueString()+"|"+mon.Target.ValueString()],
			Type:       mon.Type.ValueString(),
			Target:     mon.Target.ValueString(),
			Interval:   mon.Interval.ValueInt64(),
			TimePeriod: mon.TimePeriod.ValueInt64(),
			Alert: unifi.WANSlaAlert{
				ThresholdPolicy:  mon.ThresholdPolicy.ValueString(),
				LossThreshold:    int64Ptr(mon.LossThreshold),
				LatencyThreshold: int64Ptr(mon.LatencyThreshold),
				JitterThreshold:  int64Ptr(mon.JitterThreshold),
			},
		}
		if am.Type == unifi.WANSlaMonitorTypeDNS {
			am.Domain = mon.Domain.ValueString()
		}
		out.Monitors = append(out.Monitors, am)
	}
	return out
}

func int64Value(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func wanSlaAPIToModel(s *unifi.WANSla, m *wanSlaResourceModel) {
	m.ID = types.StringValue(s.ID)
	m.Name = types.StringValue(s.Name)
	m.MonitorHealthType = types.StringValue(s.MonitorHealthType)
	// The threshold only means something with CUSTOM; ignore whatever the controller keeps otherwise.
	if s.MonitorHealthType == unifi.WANSlaMonitorHealthCustom {
		m.MonitorDownThreshold = int64Value(s.MonitorDownThreshold)
	} else {
		m.MonitorDownThreshold = types.Int64Null()
	}
	m.Monitors = make([]wanSlaMonitorModel, 0, len(s.Monitors))
	for _, mon := range s.Monitors {
		mm := wanSlaMonitorModel{
			MonitorID:        int64Value(mon.MonitorID),
			Type:             types.StringValue(mon.Type),
			Target:           types.StringValue(mon.Target),
			Domain:           types.StringNull(),
			Interval:         types.Int64Value(mon.Interval),
			TimePeriod:       types.Int64Value(mon.TimePeriod),
			ThresholdPolicy:  types.StringValue(mon.Alert.ThresholdPolicy),
			LossThreshold:    int64Value(mon.Alert.LossThreshold),
			LatencyThreshold: int64Value(mon.Alert.LatencyThreshold),
			JitterThreshold:  int64Value(mon.Alert.JitterThreshold),
		}
		if mon.Domain != "" {
			mm.Domain = types.StringValue(mon.Domain)
		}
		m.Monitors = append(m.Monitors, mm)
	}
}
