package unifi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var (
	_ resource.Resource                = &contentFilterResource{}
	_ resource.ResourceWithImportState = &contentFilterResource{}
	_ resource.ResourceWithIdentity    = &contentFilterResource{}
)

func NewContentFilterResource() resource.Resource {
	return &contentFilterResource{}
}

// contentFilterResource manages a content filter (Settings > Security >
// Content Filtering): DNS-based category blocking per network or client.
type contentFilterResource struct {
	client *Client
}

type contentFilterModel struct {
	ID         types.String   `tfsdk:"id"`
	Site       types.String   `tfsdk:"site"`
	Name       types.String   `tfsdk:"name"`
	Enabled    types.Bool     `tfsdk:"enabled"`
	NetworkIDs types.Set      `tfsdk:"network_ids"`
	ClientMACs types.Set      `tfsdk:"client_macs"`
	Categories types.Set      `tfsdk:"categories"`
	AllowList  types.Set      `tfsdk:"allow_list"`
	BlockList  types.Set      `tfsdk:"block_list"`
	SafeSearch types.Set      `tfsdk:"safe_search"`
	Schedule   types.Object   `tfsdk:"schedule"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

type contentFilterIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

func (r *contentFilterResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_content_filter"
}

func (r *contentFilterResource) IdentitySchema(
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

func (r *contentFilterResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	strSet := func(desc string, v ...validator.Set) schema.SetAttribute {
		return schema.SetAttribute{MarkdownDescription: desc, ElementType: types.StringType, Optional: true, Validators: v}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_content_filter` manages a content filter (Settings > Security > Content " +
			"Filtering): DNS-based blocking of site categories, plus allow and block lists and safe search, " +
			"for networks or individual clients.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the content filter.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the filter with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the filter.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the filter is active. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"network_ids": strSet("IDs of the networks the filter applies to."),
			"client_macs": strSet("MAC addresses of the clients the filter applies to."),
			"categories": strSet("Blocked categories, e.g. `ADVERTISEMENT`, `MALWARE`, `PHISHING`, " +
				"`GAMBLING`, `ADULT`, or the presets `FAMILY` and `WORK`."),
			"allow_list": strSet("Domains that are always allowed."),
			"block_list": strSet("Domains that are always blocked."),
			"safe_search": strSet("Search engines forced into safe search: `GOOGLE`, `YOUTUBE`, `BING`.",
				setvalidator.ValueStringsAre(stringvalidator.OneOf("GOOGLE", "YOUTUBE", "BING"))),
			"schedule": ruleScheduleSchemaAttribute(),
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
		},
	}
}

func (r *contentFilterResource) Configure(
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

func (r *contentFilterResource) site(m *contentFilterModel) string {
	if s := m.Site.ValueString(); s != "" {
		return s
	}
	return r.client.Site
}

func (r *contentFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contentFilterModel
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
	f := r.modelToAPI(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateContentFiltering(ctx, site, f)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Content Filter", err.Error())
		return
	}
	r.apiToModel(ctx, created, &plan, site, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, contentFilterIdentityModel{ID: plan.ID, Site: types.StringValue(site)})...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contentFilterModel
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

	var identity contentFilterIdentityModel
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

	f, err := r.client.GetContentFiltering(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Content Filter", "Could not read content filter "+id+": "+err.Error())
		return
	}
	r.apiToModel(ctx, f, &state, site, &resp.Diagnostics)
	if req.Identity == nil || req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, contentFilterIdentityModel{ID: state.ID, Site: types.StringValue(site)})...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contentFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state contentFilterModel
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
	f := r.modelToAPI(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	f.ID = state.ID.ValueString()
	updated, err := r.client.UpdateContentFiltering(ctx, site, f)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Content Filter", err.Error())
		return
	}
	r.apiToModel(ctx, updated, &plan, site, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state contentFilterModel
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

	err := r.client.DeleteContentFiltering(ctx, r.site(&state), state.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Content Filter", err.Error())
	}
}

func (r *contentFilterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	id, site := req.ID, r.client.Site
	if id == "" {
		var identity contentFilterIdentityModel
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
	resp.Diagnostics.Append(resp.Identity.Set(ctx, contentFilterIdentityModel{
		ID: types.StringValue(id), Site: types.StringValue(site),
	})...)
}

func (r *contentFilterResource) modelToAPI(
	ctx context.Context,
	m *contentFilterModel,
	diags *diag.Diagnostics,
) *unifi.ContentFiltering {
	s := scheduleFromObject(ctx, m.Schedule, diags)
	return &unifi.ContentFiltering{
		Name:       m.Name.ValueString(),
		Enabled:    m.Enabled.ValueBool(),
		NetworkIDs: stringsFromSet(ctx, m.NetworkIDs, diags),
		ClientMACs: stringsFromSet(ctx, m.ClientMACs, diags),
		Categories: stringsFromSet(ctx, m.Categories, diags),
		AllowList:  stringsFromSet(ctx, m.AllowList, diags),
		BlockList:  stringsFromSet(ctx, m.BlockList, diags),
		SafeSearch: stringsFromSet(ctx, m.SafeSearch, diags),
		Schedule: &unifi.ContentFilteringSchedule{
			Mode: s.Mode, RepeatOnDays: s.RepeatOnDays, TimeAllDay: s.TimeAllDay,
			TimeRangeStart: s.TimeRangeStart, TimeRangeEnd: s.TimeRangeEnd,
			Date: s.Date, DateStart: s.DateStart, DateEnd: s.DateEnd,
		},
	}
}

func (r *contentFilterResource) apiToModel(
	ctx context.Context,
	f *unifi.ContentFiltering,
	m *contentFilterModel,
	site string,
	diags *diag.Diagnostics,
) {
	m.ID = types.StringValue(f.ID)
	m.Site = types.StringValue(site)
	m.Name = types.StringValue(f.Name)
	m.Enabled = types.BoolValue(f.Enabled)
	m.NetworkIDs = setOrNull(ctx, f.NetworkIDs, diags)
	m.ClientMACs = setOrNull(ctx, f.ClientMACs, diags)
	m.Categories = setOrNull(ctx, f.Categories, diags)
	m.AllowList = setOrNull(ctx, f.AllowList, diags)
	m.BlockList = setOrNull(ctx, f.BlockList, diags)
	m.SafeSearch = setOrNull(ctx, f.SafeSearch, diags)
	var s ruleSchedule
	if f.Schedule != nil {
		s = ruleSchedule{
			Mode: f.Schedule.Mode, RepeatOnDays: f.Schedule.RepeatOnDays,
			TimeAllDay: f.Schedule.TimeAllDay, TimeRangeStart: f.Schedule.TimeRangeStart,
			TimeRangeEnd: f.Schedule.TimeRangeEnd, Date: f.Schedule.Date,
			DateStart: f.Schedule.DateStart, DateEnd: f.Schedule.DateEnd,
		}
	}
	m.Schedule = scheduleToObject(ctx, s, diags)
}
