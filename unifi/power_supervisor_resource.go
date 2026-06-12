package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &powerSupervisorResource{}
	_ resource.ResourceWithImportState = &powerSupervisorResource{}
)

func NewPowerSupervisorResource() resource.Resource {
	return &powerSupervisorResource{}
}

// powerSupervisorResource defines the resource implementation.
type powerSupervisorResource struct {
	client *Client
}

// powerSupervisorResourceModel describes the resource data model.
type powerSupervisorResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Site                types.String `tfsdk:"site"`
	DeviceMAC           types.String `tfsdk:"device_mac"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	HeartbeatInterval   types.Int64  `tfsdk:"heartbeat_interval"`
	SilenceThreshold    types.Int64  `tfsdk:"silence_threshold"`
	PowerOffDuration    types.Int64  `tfsdk:"power_off_duration"`
	ConsecutiveFailures types.Int64  `tfsdk:"consecutive_failures"`
	PowerSources        types.List   `tfsdk:"power_sources"`
}

// powerSourceAttrTypes is the object schema of a resolved upstream power source.
func powerSourceAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"client_psu_index":   types.Int64Type,
		"power_source_index": types.Int64Type,
		"power_source_mac":   types.StringType,
		"power_source_type":  types.StringType,
	}
}

func (r *powerSupervisorResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_power_supervisor"
}

func (r *powerSupervisorResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a UniFi **Device Supervisor** (UniFi Network 10.2+): " +
			"heartbeat monitoring of a device plus automatic power-cycling of its upstream " +
			"PoE source after a silence threshold. The supervised device is referenced by " +
			"its MAC; the controller resolves the upstream PoE port automatically.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The controller-assigned ID of the power supervisor.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the supervisor belongs to.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_mac": schema.StringAttribute{
				MarkdownDescription: "MAC address of the supervised device (the controller keys " +
					"supervisors per device). Changing it replaces the supervisor.",
				Required: true,
				Validators: []validator.String{
					validators.MACAddressValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether supervision is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"heartbeat_interval": schema.Int64Attribute{
				MarkdownDescription: "How often (seconds) the controller probes the device. " +
					"Defaults to `60`.",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(60),
			},
			"silence_threshold": schema.Int64Attribute{
				MarkdownDescription: "How long (seconds) the device may be silent before the " +
					"controller power-cycles its upstream PoE source. Defaults to `900` (15 min).",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(900),
			},
			"power_off_duration": schema.Int64Attribute{
				MarkdownDescription: "How long (seconds) the upstream PoE source stays off during " +
					"a power-cycle. Defaults to `120` (2 min).",
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(120),
			},
			"consecutive_failures": schema.Int64Attribute{
				MarkdownDescription: "Number of consecutive heartbeat failures observed by the " +
					"controller (read-only).",
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"power_sources": schema.ListNestedAttribute{
				MarkdownDescription: "The upstream power source(s) the controller resolved for the " +
					"device and will cycle on recovery (read-only).",
				Computed: true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"client_psu_index": schema.Int64Attribute{
							MarkdownDescription: "Index of the supervised device's PSU.",
							Computed:            true,
						},
						"power_source_index": schema.Int64Attribute{
							MarkdownDescription: "Port/outlet index on the upstream source.",
							Computed:            true,
						},
						"power_source_mac": schema.StringAttribute{
							MarkdownDescription: "MAC of the upstream source (e.g. the PoE switch).",
							Computed:            true,
						},
						"power_source_type": schema.StringAttribute{
							MarkdownDescription: "Type of the upstream source (e.g. `poe_port`).",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (r *powerSupervisorResource) Configure(
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

func (r *powerSupervisorResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data powerSupervisorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	created, err := r.client.CreatePowerSupervisor(ctx, site, r.modelToPowerSupervisor(&data))
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Power Supervisor", err.Error())
		return
	}

	resp.Diagnostics.Append(r.powerSupervisorToModel(created, &data, site)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *powerSupervisorResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data powerSupervisorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	supervisor, err := r.client.GetPowerSupervisor(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Power Supervisor",
			"Could not read power supervisor with ID "+data.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(r.powerSupervisorToModel(supervisor, &data, site)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *powerSupervisorResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data powerSupervisorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	supervisor := r.modelToPowerSupervisor(&data)
	supervisor.ID = data.ID.ValueString()

	updated, err := r.client.UpdatePowerSupervisor(ctx, site, supervisor)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Power Supervisor", err.Error())
		return
	}

	resp.Diagnostics.Append(r.powerSupervisorToModel(updated, &data, site)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *powerSupervisorResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data powerSupervisorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	err := r.client.DeletePowerSupervisor(ctx, site, data.ID.ValueString())
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			return
		}
		resp.Diagnostics.AddError("Error Deleting Power Supervisor", err.Error())
		return
	}
}

func (r *powerSupervisorResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Accept "site:id", "site:mac", a bare controller id, or the supervised
	// device's MAC. A MAC contains colons, so a bare MAC must be detected before
	// splitting on ":" (otherwise "9c:05:..." would parse as site "9c").
	macRE := regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
	site := r.client.Site
	identifier := req.ID
	if !macRE.MatchString(req.ID) {
		if parts := strings.SplitN(req.ID, ":", 2); len(parts) == 2 {
			site = parts[0]
			identifier = parts[1]
		}
	}

	// A MAC address identifies the supervised device; resolve it to the
	// controller id (the collection is keyed per device).
	if macRE.MatchString(identifier) {
		supervisor, err := r.client.GetPowerSupervisorByMAC(ctx, site, strings.ToLower(identifier))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Importing Power Supervisor",
				"Could not find a power supervisor for device MAC "+identifier+": "+err.Error(),
			)
			return
		}
		identifier = supervisor.ID
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), identifier)...)
}

// modelToPowerSupervisor converts the Terraform model to the API struct. The
// controller resolves power_sources, so they are not sent.
func (r *powerSupervisorResource) modelToPowerSupervisor(
	model *powerSupervisorResourceModel,
) *unifi.PowerSupervisor {
	return &unifi.PowerSupervisor{
		ClientMAC: model.DeviceMAC.ValueString(),
		Enabled:   model.Enabled.ValueBool(),
		Settings: unifi.PowerSupervisorSettings{
			HeartbeatInterval: int(model.HeartbeatInterval.ValueInt64()),
			SilenceThreshold:  int(model.SilenceThreshold.ValueInt64()),
			PowerOffDuration:  int(model.PowerOffDuration.ValueInt64()),
		},
		PowerSources: []unifi.PowerSupervisorSource{},
	}
}

// powerSupervisorToModel converts the API struct to the Terraform model.
func (r *powerSupervisorResource) powerSupervisorToModel(
	supervisor *unifi.PowerSupervisor,
	model *powerSupervisorResourceModel,
	site string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(supervisor.ID)
	model.Site = types.StringValue(site)
	model.DeviceMAC = types.StringValue(supervisor.ClientMAC)
	model.Enabled = types.BoolValue(supervisor.Enabled)
	model.HeartbeatInterval = types.Int64Value(int64(supervisor.Settings.HeartbeatInterval))
	model.SilenceThreshold = types.Int64Value(int64(supervisor.Settings.SilenceThreshold))
	model.PowerOffDuration = types.Int64Value(int64(supervisor.Settings.PowerOffDuration))
	model.ConsecutiveFailures = types.Int64Value(int64(supervisor.ConsecutiveFailures))

	elements := make([]attr.Value, 0, len(supervisor.PowerSources))
	for _, src := range supervisor.PowerSources {
		obj, objDiags := types.ObjectValue(powerSourceAttrTypes(), map[string]attr.Value{
			"client_psu_index":   types.Int64Value(int64(src.ClientPsuIndex)),
			"power_source_index": types.Int64Value(int64(src.PowerSourceIndex)),
			"power_source_mac":   types.StringValue(src.PowerSourceMAC),
			"power_source_type":  types.StringValue(src.PowerSourceType),
		})
		diags.Append(objDiags...)
		elements = append(elements, obj)
	}

	list, listDiags := types.ListValue(
		types.ObjectType{AttrTypes: powerSourceAttrTypes()},
		elements,
	)
	diags.Append(listDiags...)
	model.PowerSources = list

	return diags
}
