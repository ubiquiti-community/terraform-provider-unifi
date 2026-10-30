package unifi

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// devicePortResource manages a single switch port's overrides as its own
// resource, keyed by "<device_mac>/<index>". Unlike unifi_device's
// port_override SetNestedBlock, which has no per-element import ID, this
// resource can be imported directly. It is not compatible with declaring a
// port_override block for the same index on the same unifi_device - see the
// note on both resources' schema descriptions below.

var (
	_ resource.Resource                = &devicePortResource{}
	_ resource.ResourceWithImportState = &devicePortResource{}
)

func NewDevicePortResource() resource.Resource {
	return &devicePortResource{}
}

type devicePortResource struct {
	client *Client
}

// devicePortResourceModel mirrors portOverrideModel (device_resource.go) plus
// its own identity: device_mac + index instead of living inside unifi_device.
type devicePortResourceModel struct {
	ID                         types.String         `tfsdk:"id"`
	Site                       types.String         `tfsdk:"site"`
	DeviceMAC                  hwtypes.MACAddress   `tfsdk:"device_mac"`
	Index                      types.Int64          `tfsdk:"index"`
	Name                       types.String         `tfsdk:"name"`
	PortProfileID              types.String         `tfsdk:"port_profile_id"`
	OpMode                     types.String         `tfsdk:"op_mode"`
	PoeMode                    types.String         `tfsdk:"poe_mode"`
	AggregateMembers           types.List           `tfsdk:"aggregate_members"`
	Autoneg                    types.Bool           `tfsdk:"autoneg"`
	Dot1XCtrl                  types.String         `tfsdk:"dot1x_ctrl"`
	Dot1XIDleTimeout           timetypes.GoDuration `tfsdk:"dot1x_idle_timeout"`
	EgressRateLimitKbps        types.Int64          `tfsdk:"egress_rate_limit_kbps"`
	EgressRateLimitKbpsEnabled types.Bool           `tfsdk:"egress_rate_limit_kbps_enabled"`
	ExcludedNetworkIDs         types.Set            `tfsdk:"excluded_networkconf_ids"`
	FecMode                    types.String         `tfsdk:"fec_mode"`
	FlowControlEnabled         types.Bool           `tfsdk:"flow_control_enabled"`
	Forward                    types.String         `tfsdk:"forward"`
	FullDuplex                 types.Bool           `tfsdk:"full_duplex"`
	Isolation                  types.Bool           `tfsdk:"isolation"`
	LldpmedEnabled             types.Bool           `tfsdk:"lldpmed_enabled"`
	LldpmedNotifyEnabled       types.Bool           `tfsdk:"lldpmed_notify_enabled"`
	MirrorPortIDX              types.Int64          `tfsdk:"mirror_port_idx"`
	MulticastRouterNetworkIDs  types.Set            `tfsdk:"multicast_router_networkconf_ids"`
	NativeNetworkID            types.String         `tfsdk:"native_networkconf_id"`
	PortKeepaliveEnabled       types.Bool           `tfsdk:"port_keepalive_enabled"`
	PortSecurityEnabled        types.Bool           `tfsdk:"port_security_enabled"`
	PortSecurityMACAddress     types.List           `tfsdk:"port_security_mac_address"`
	PriorityQueue1Level        types.Int64          `tfsdk:"priority_queue1_level"`
	PriorityQueue2Level        types.Int64          `tfsdk:"priority_queue2_level"`
	PriorityQueue3Level        types.Int64          `tfsdk:"priority_queue3_level"`
	PriorityQueue4Level        types.Int64          `tfsdk:"priority_queue4_level"`
	SettingPreference          types.String         `tfsdk:"setting_preference"`
	Speed                      types.Int64          `tfsdk:"speed"`
	StormctrlBroadcastEnabled  types.Bool           `tfsdk:"stormctrl_bcast_enabled"`
	StormctrlBroadcastLevel    types.Int64          `tfsdk:"stormctrl_bcast_level"`
	StormctrlBroadcastRate     types.Int64          `tfsdk:"stormctrl_bcast_rate"`
	StormctrlMcastEnabled      types.Bool           `tfsdk:"stormctrl_mcast_enabled"`
	StormctrlMcastLevel        types.Int64          `tfsdk:"stormctrl_mcast_level"`
	StormctrlMcastRate         types.Int64          `tfsdk:"stormctrl_mcast_rate"`
	StormctrlType              types.String         `tfsdk:"stormctrl_type"`
	StormctrlUcastEnabled      types.Bool           `tfsdk:"stormctrl_ucast_enabled"`
	StormctrlUcastLevel        types.Int64          `tfsdk:"stormctrl_ucast_level"`
	StormctrlUcastRate         types.Int64          `tfsdk:"stormctrl_ucast_rate"`
	StpPortMode                types.Bool           `tfsdk:"stp_port_mode"`
	TaggedNetworkIDs           types.Set            `tfsdk:"tagged_networkconf_ids"`
	TaggedVLANMgmt             types.String         `tfsdk:"tagged_vlan_mgmt"`
	VoiceNetworkID             types.String         `tfsdk:"voice_networkconf_id"`
}

func (r *devicePortResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_device_port"
}

func (r *devicePortResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a single switch port's overrides as its own resource, " +
			"identified by `device_mac` + `index` (id `\"<device_mac>/<index>\"`). Unlike " +
			"`port_override` (a SetNestedBlock, which has no per-element import ID), this " +
			"resource can be imported directly with a plain `terraform import` or `import` " +
			"block. Internally it does a read-modify-write against the parent device's full " +
			"port_overrides array (the UniFi API has no per-port endpoint), touching only the " +
			"declared index and leaving every other port's provider-modeled settings as the " +
			"controller already has it.\n\n" +
			"**Caveat:** the underlying API client does not model every field the controller " +
			"returns per port (for example `stp_edge_state`). Any write through this resource " +
			"sends the whole port_overrides array reconstructed from modeled fields only, so " +
			"unmodeled fields are not round-tripped and can reset to their default - both on the " +
			"declared port itself and on every other port in the array. This is a pre-existing " +
			"limitation of the shared read-modify-write path, not specific to this resource - " +
			"`port_override` below has the same exposure whenever it writes a change.\n\n" +
			"A port only appears in the controller's port_overrides array once it has been " +
			"customized at least once (the same \"nothing to override\" ports `port_override` " +
			"describes). A never-customized port therefore cannot be read or imported here " +
			"either - only ports that already carry some override are visible.\n\n" +
			"**Note:** not compatible with the [`port_override`](device.md#nestedblock--port_override) " +
			"block on `unifi_device` - do not declare both for the same `device_mac` + `index`. " +
			"The two write to the same controller-side field independently, so whichever applies " +
			"last wins and the other's plan immediately shows drift. Manage a given port through " +
			"exactly one of the two.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "\"<device_mac>/<index>\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "Site name. Defaults to the provider's site.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"device_mac": schema.StringAttribute{
				Description: "MAC address of the parent device.",
				CustomType:  hwtypes.MACAddressType{},
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"index": schema.Int64Attribute{
				Description: "Switch port index.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name of the port.",
				Optional:    true,
			},
			"port_profile_id": schema.StringAttribute{
				Description: "ID of the Port Profile used on this port.",
				Optional:    true,
			},
			"op_mode": schema.StringAttribute{
				// Deliberately no Default: a StaticString("switch") default
				// makes an omitted op_mode indistinguishable from an
				// explicit `op_mode = "switch"` (both resolve to the same
				// known "switch" plan value), so modelToAPIPortOverride
				// could no longer tell "user didn't mention it" from "user
				// wants it reverted to switch" - the same silent-write bug
				// class the resource exists to avoid for the boolean
				// attributes below. Like those, an omitted op_mode simply
				// plans as unknown and modelToAPIPortOverride falls back to
				// the port's current controller value.
				Description: "Operating mode of the port: `switch` (default), `mirror`, or `aggregate`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("switch", "mirror", "aggregate"),
				},
			},
			"poe_mode": schema.StringAttribute{
				Description: "PoE mode of the port; valid values are `auto`, `pasv24`, `passthrough`, and `off`.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("auto", "pasv24", "passthrough", "off"),
				},
			},
			"aggregate_members": schema.ListAttribute{
				Description: "Port indices that make up this link-aggregation (LAG) group.",
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"autoneg": schema.BoolAttribute{
				Description: "Enable auto-negotiation for port speed.",
				Optional:    true,
				Computed:    true,
			},
			"dot1x_ctrl": schema.StringAttribute{
				Description: "802.1X control mode.",
				Optional:    true,
			},
			"dot1x_idle_timeout": schema.StringAttribute{
				Description: "802.1X idle timeout, as a Go duration string (e.g. `5m`, `300s`).",
				CustomType:  timetypes.GoDurationType{},
				Optional:    true,
				Validators: []validator.String{
					validators.GoDurationBetween(0, 65535*time.Second),
					validators.GoDurationMultipleOf(time.Second),
				},
			},
			"egress_rate_limit_kbps": schema.Int64Attribute{
				Description: "Egress rate limit in kbps.",
				Optional:    true,
			},
			"egress_rate_limit_kbps_enabled": schema.BoolAttribute{
				Description: "Enable egress rate limiting.",
				Optional:    true,
				Computed:    true,
			},
			"excluded_networkconf_ids": schema.SetAttribute{
				Description: "List of network IDs to exclude from this port.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"fec_mode": schema.StringAttribute{
				Description: "Forward Error Correction mode.",
				Optional:    true,
			},
			"flow_control_enabled": schema.BoolAttribute{
				Description: "Enable flow control.",
				Optional:    true,
				Computed:    true,
			},
			"forward": schema.StringAttribute{
				Description: "Forwarding mode.",
				Optional:    true,
			},
			"full_duplex": schema.BoolAttribute{
				Description: "Enable full duplex mode.",
				Optional:    true,
				Computed:    true,
			},
			"isolation": schema.BoolAttribute{
				Description: "Enable port isolation.",
				Optional:    true,
				Computed:    true,
			},
			"lldpmed_enabled": schema.BoolAttribute{
				Description: "Enable LLDP-MED.",
				Optional:    true,
				Computed:    true,
			},
			"lldpmed_notify_enabled": schema.BoolAttribute{
				Description: "Enable LLDP-MED notifications.",
				Optional:    true,
				Computed:    true,
			},
			"mirror_port_idx": schema.Int64Attribute{
				Description: "Mirror port index.",
				Optional:    true,
			},
			"multicast_router_networkconf_ids": schema.SetAttribute{
				Description: "List of network IDs for multicast router.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"native_networkconf_id": schema.StringAttribute{
				Description: "Native network ID (VLAN).",
				Optional:    true,
			},
			"port_keepalive_enabled": schema.BoolAttribute{
				Description: "Enable port keepalive.",
				Optional:    true,
				Computed:    true,
			},
			"port_security_enabled": schema.BoolAttribute{
				Description: "Enable port security.",
				Optional:    true,
				Computed:    true,
			},
			"port_security_mac_address": schema.ListAttribute{
				Description: "List of MAC addresses allowed when port security is enabled.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"priority_queue1_level": schema.Int64Attribute{
				Description: "Priority queue 1 level.",
				Optional:    true,
			},
			"priority_queue2_level": schema.Int64Attribute{
				Description: "Priority queue 2 level.",
				Optional:    true,
			},
			"priority_queue3_level": schema.Int64Attribute{
				Description: "Priority queue 3 level.",
				Optional:    true,
			},
			"priority_queue4_level": schema.Int64Attribute{
				Description: "Priority queue 4 level.",
				Optional:    true,
			},
			"setting_preference": schema.StringAttribute{
				Description: "Setting preference.",
				Optional:    true,
			},
			"speed": schema.Int64Attribute{
				Description: "Port speed in Mbps.",
				Optional:    true,
			},
			"stormctrl_bcast_enabled": schema.BoolAttribute{
				Description: "Enable broadcast storm control.",
				Optional:    true,
				Computed:    true,
			},
			"stormctrl_bcast_level": schema.Int64Attribute{
				Description: "Broadcast storm control level.",
				Optional:    true,
			},
			"stormctrl_bcast_rate": schema.Int64Attribute{
				Description: "Broadcast storm control rate.",
				Optional:    true,
			},
			"stormctrl_mcast_enabled": schema.BoolAttribute{
				Description: "Enable multicast storm control.",
				Optional:    true,
				Computed:    true,
			},
			"stormctrl_mcast_level": schema.Int64Attribute{
				Description: "Multicast storm control level.",
				Optional:    true,
			},
			"stormctrl_mcast_rate": schema.Int64Attribute{
				Description: "Multicast storm control rate.",
				Optional:    true,
			},
			"stormctrl_type": schema.StringAttribute{
				Description: "Storm control type.",
				Optional:    true,
			},
			"stormctrl_ucast_enabled": schema.BoolAttribute{
				Description: "Enable unicast storm control.",
				Optional:    true,
				Computed:    true,
			},
			"stormctrl_ucast_level": schema.Int64Attribute{
				Description: "Unicast storm control level.",
				Optional:    true,
			},
			"stormctrl_ucast_rate": schema.Int64Attribute{
				Description: "Unicast storm control rate.",
				Optional:    true,
			},
			"stp_port_mode": schema.BoolAttribute{
				Description: "STP port mode.",
				Optional:    true,
				Computed:    true,
			},
			"tagged_networkconf_ids": schema.SetAttribute{
				Description: "List of network IDs to tag on this port.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tagged_vlan_mgmt": schema.StringAttribute{
				Description: "Tagged VLAN management.",
				Optional:    true,
			},
			"voice_networkconf_id": schema.StringAttribute{
				Description: "Voice network ID.",
				Optional:    true,
			},
		},
	}
}

func (r *devicePortResource) Configure(
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

func (r *devicePortResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan devicePortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags := r.upsert(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

func (r *devicePortResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan devicePortResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, diags := r.upsert(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, result)...)
}

// upsert reads the parent device, splices the declared port into its
// port_overrides array (mergePortOverridesByIndex, shared with unifi_device),
// PUTs the device back, and returns state built from the port entry the
// controller actually stored.
func (r *devicePortResource) upsert(
	ctx context.Context,
	plan devicePortResourceModel,
) (devicePortResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	mac := cleanMAC(plan.DeviceMAC.ValueString())
	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}
	index := plan.Index.ValueInt64()

	// unifi_device_port has no per-port API endpoint: it reads the whole
	// device, splices in the declared port, and PUTs the device back. Without
	// this lock, two unifi_device_port resources for different indices on the
	// same device applying concurrently (default parallelism) can both read
	// before either writes, and whichever PUT lands second silently discards
	// the other's change.
	unlock := r.client.lockDevice(mac)
	defer unlock()

	currentDevice, err := r.client.GetDeviceByMAC(ctx, site, mac)
	if err != nil {
		diags.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device with MAC %s: %s", mac, err),
		)
		return plan, diags
	}
	if currentDevice == nil {
		diags.AddError("Device Not Found", fmt.Sprintf("Device not found using mac %s", mac))
		return plan, diags
	}

	var currentPO unifi.DevicePortOverrides
	exists := false
	for _, po := range currentDevice.PortOverrides {
		if po.PortIDX != nil && *po.PortIDX == index {
			currentPO = po
			exists = true
			break
		}
	}
	if !exists {
		diags.AddError(
			"Port Not Found",
			fmt.Sprintf(
				"Device %s has no port at index %d in its current port_overrides; "+
					"only existing physical ports can be managed.",
				mac, index,
			),
		)
		return plan, diags
	}

	desiredPO, convDiags := modelToAPIPortOverride(ctx, plan, currentPO)
	diags.Append(convDiags...)
	if diags.HasError() {
		return plan, diags
	}

	currentDevice.PortOverrides = mergePortOverridesByIndex(
		currentDevice.PortOverrides, []unifi.DevicePortOverrides{desiredPO},
	)

	_, err = r.client.UpdateDevice(ctx, site, currentDevice)
	if err != nil {
		diags.AddError(
			"Error Updating Device Port",
			fmt.Sprintf("Could not update port %d on device %s: %s", index, mac, err),
		)
		return plan, diags
	}

	// UpdateDevice's PUT provisions asynchronously: the response it returns can
	// still reflect pre-update port_overrides while the change applies on the
	// device. Trusting it directly risks recording stale state, or a false
	// "Port Missing After Update" if the port briefly drops out mid-transition.
	// Wait for the device to settle back to connected and re-read it instead -
	// mirrors deviceResource.updateDevice's own post-update wait.
	updatedDevice, err := waitForDeviceState(
		ctx,
		r.client,
		site, mac,
		unifi.DeviceStateConnected,
		[]unifi.DeviceState{unifi.DeviceStateAdopting, unifi.DeviceStateProvisioning},
		3*time.Minute,
	)
	if err != nil {
		diags.AddError(
			"Error Waiting for Device Port Update",
			fmt.Sprintf(
				"Could not wait for device %s to settle after updating port %d: %s",
				mac,
				index,
				err,
			),
		)
		return plan, diags
	}

	result := plan
	result.ID = types.StringValue(fmt.Sprintf("%s/%d", mac, index))
	result.Site = types.StringValue(site)
	result.DeviceMAC = hwtypes.NewMACAddressValue(mac)

	for _, po := range updatedDevice.PortOverrides {
		if po.PortIDX != nil && *po.PortIDX == index {
			diags.Append(applyAPIPortOverrideToDevicePortModel(&result, po)...)
			return result, diags
		}
	}

	diags.AddError(
		"Port Missing After Update",
		fmt.Sprintf("Device %s no longer reports a port at index %d after the update.", mac, index),
	)
	return result, diags
}

func (r *devicePortResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state devicePortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mac := cleanMAC(state.DeviceMAC.ValueString())
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}
	index := state.Index.ValueInt64()

	// Same lock as upsert/updateDevice/the port action: without it, a refresh
	// can read the device while a sibling writer is mid-provisioning and
	// observe a transient response missing this port, which the loop below
	// would treat as a permanent removal (RemoveResource) instead of retrying
	// once the write settles.
	unlock := r.client.lockDevice(mac)
	defer unlock()

	device, err := r.client.GetDeviceByMAC(ctx, site, mac)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device with MAC %s: %s", mac, err),
		)
		return
	}
	if device == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	for _, po := range device.PortOverrides {
		if po.PortIDX != nil && *po.PortIDX == index {
			state.ID = types.StringValue(fmt.Sprintf("%s/%d", mac, index))
			state.Site = types.StringValue(site)
			resp.Diagnostics.Append(applyAPIPortOverrideToDevicePortModel(&state, po)...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
			return
		}
	}

	// Port index no longer present on the device (e.g. a module removed).
	resp.State.RemoveResource(ctx)
}

func (r *devicePortResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state devicePortResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mirrors port_override's own removal semantics (see its block description
	// in device_resource.go): stop managing the port, but don't attempt to
	// reset it on the controller.
	resp.Diagnostics.AddWarning(
		"Port Left As-Is",
		fmt.Sprintf(
			"unifi_device_port only stops managing index %d on %s; the controller's current "+
				"settings for it are left untouched.",
			state.Index.ValueInt64(), state.DeviceMAC.ValueString(),
		),
	)
}

func (r *devicePortResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	mac, index, err := parseDevicePortImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("device_mac"), mac)...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("index"), index)...)
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s/%d", mac, index))...)
}

// parseDevicePortImportID splits an import ID of the form
// "<device_mac>/<index>" into its parts. A slash separator (rather than a
// colon) sidesteps having to split on the MAC's own colon-separated octets:
// "a8:9c:6c:08:ea:3b/2" -> mac "a8:9c:6c:08:ea:3b", index 2.
func parseDevicePortImportID(importID string) (mac string, index int64, err error) {
	parts := strings.SplitN(importID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", 0, fmt.Errorf(
			`import ID must be "<device_mac>/<index>", e.g. "a8:9c:6c:08:ea:3b/2", got %q`,
			importID,
		)
	}

	index, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("could not parse port index %q as an integer: %w", parts[1], err)
	}

	return cleanMAC(parts[0]), index, nil
}

// applyAPIPortOverrideToDevicePortModel copies an API port override onto a
// devicePortResourceModel, reusing device_resource.go's apiPortOverrideToModel
// (portOverrideModel has the same field set minus device_mac/site/id).
func applyAPIPortOverrideToDevicePortModel(
	model *devicePortResourceModel,
	po unifi.DevicePortOverrides,
) diag.Diagnostics {
	pom, diags := apiPortOverrideToModel(po)
	if diags.HasError() {
		return diags
	}

	model.Index = pom.Index
	model.Name = pom.Name
	model.PortProfileID = pom.PortProfileID
	// apiPortOverrideToModel nulls OpMode when the API reports "" (the
	// switch/default mode is never sent, so this is the common case - see
	// modelToAPIPortOverride). Unlike the other empty-string-as-null fields
	// above, op_mode is never truly absent on a live port - the OneOf
	// validator only allows "switch"/"mirror"/"aggregate" - and this
	// resource has no schema Default to reconcile it with a config value.
	// Leaving it null here would make Terraform's post-apply consistency
	// check fail the moment a user explicitly declares `op_mode = "switch"`
	// on a port that reports "" (state null != config "switch"). Represent
	// the default mode as the literal string instead.
	if po.OpMode == "" {
		model.OpMode = types.StringValue("switch")
	} else {
		model.OpMode = pom.OpMode
	}
	model.PoeMode = pom.PoeMode
	model.AggregateMembers = pom.AggregateMembers
	model.Autoneg = pom.Autoneg
	model.Dot1XCtrl = pom.Dot1XCtrl
	model.Dot1XIDleTimeout = pom.Dot1XIDleTimeout
	model.EgressRateLimitKbps = pom.EgressRateLimitKbps
	model.EgressRateLimitKbpsEnabled = pom.EgressRateLimitKbpsEnabled
	model.ExcludedNetworkIDs = pom.ExcludedNetworkIDs
	model.FecMode = pom.FecMode
	model.FlowControlEnabled = pom.FlowControlEnabled
	model.Forward = pom.Forward
	model.FullDuplex = pom.FullDuplex
	model.Isolation = pom.Isolation
	model.LldpmedEnabled = pom.LldpmedEnabled
	model.LldpmedNotifyEnabled = pom.LldpmedNotifyEnabled
	model.MirrorPortIDX = pom.MirrorPortIDX
	model.MulticastRouterNetworkIDs = pom.MulticastRouterNetworkIDs
	model.NativeNetworkID = pom.NativeNetworkID
	model.PortKeepaliveEnabled = pom.PortKeepaliveEnabled
	model.PortSecurityEnabled = pom.PortSecurityEnabled
	model.PortSecurityMACAddress = pom.PortSecurityMACAddress
	model.PriorityQueue1Level = pom.PriorityQueue1Level
	model.PriorityQueue2Level = pom.PriorityQueue2Level
	model.PriorityQueue3Level = pom.PriorityQueue3Level
	model.PriorityQueue4Level = pom.PriorityQueue4Level
	model.SettingPreference = pom.SettingPreference
	model.Speed = pom.Speed
	model.StormctrlBroadcastEnabled = pom.StormctrlBroadcastEnabled
	model.StormctrlBroadcastLevel = pom.StormctrlBroadcastLevel
	model.StormctrlBroadcastRate = pom.StormctrlBroadcastRate
	model.StormctrlMcastEnabled = pom.StormctrlMcastEnabled
	model.StormctrlMcastLevel = pom.StormctrlMcastLevel
	model.StormctrlMcastRate = pom.StormctrlMcastRate
	model.StormctrlType = pom.StormctrlType
	model.StormctrlUcastEnabled = pom.StormctrlUcastEnabled
	model.StormctrlUcastLevel = pom.StormctrlUcastLevel
	model.StormctrlUcastRate = pom.StormctrlUcastRate
	model.StpPortMode = pom.StpPortMode
	model.TaggedNetworkIDs = pom.TaggedNetworkIDs
	model.TaggedVLANMgmt = pom.TaggedVLANMgmt
	model.VoiceNetworkID = pom.VoiceNetworkID

	return diags
}

// boolOrCurrent resolves an Optional+Computed bool attribute: when it is
// unset in config (null or, absent a UseStateForUnknown plan modifier,
// unknown), it falls back to current rather than the Go zero value false.
func boolOrCurrent(v types.Bool, current bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return current
	}
	return v.ValueBool()
}

// modelToAPIPortOverride is the reverse of applyAPIPortOverrideToDevicePortModel,
// mirroring the per-element body of frameworkToPortOverrides (device_resource.go)
// but reading directly from typed model fields instead of a types.Object.
// current is the port's existing controller-side override (from the device
// read in upsert), used to resolve Optional+Computed attributes the plan
// left unknown/null instead of sending their Go zero value.
func modelToAPIPortOverride(
	ctx context.Context,
	model devicePortResourceModel,
	current unifi.DevicePortOverrides,
) (unifi.DevicePortOverrides, diag.Diagnostics) {
	var diags diag.Diagnostics

	po := unifi.DevicePortOverrides{
		PortIDX: model.Index.ValueInt64Pointer(),
		// qos_profile has no schema attribute (not modeled at all by this
		// resource), so there is no config value to ever send here - carry
		// the controller's current value forward unconditionally, or an
		// upsert of a port that has a QoS Profile assigned silently clears it
		// (mergePortOverridesByIndex fully replaces the matched entry with
		// this struct; carryUnwritableFields only forward-carries op_mode).
		QOSProfile: current.QOSProfile,
	}

	if !model.Name.IsNull() {
		po.Name = model.Name.ValueString()
	}
	if !model.PortProfileID.IsNull() {
		po.PortProfileID = model.PortProfileID.ValueString()
	}
	// See device_resource.go's frameworkToPortOverrides: op_mode is only sent
	// when non-default, since UDM gateways reject it on update (#213); gateway
	// ports never carry a non-default op_mode on the controller, so the second
	// branch below never fires for them. An explicitly declared op_mode =
	// "switch" is therefore normally a no-op, EXCEPT when the port is
	// currently mirror/aggregate on the controller - there, "switch" must
	// still be sent to actually revert it, or carryUnwritableFields
	// (device_resource.go) will re-carry the current non-default value forward
	// forever. Both branches require IsNull()/IsUnknown() to be false: op_mode
	// has no schema Default (see the schema comment), so an omitted op_mode
	// is Unknown, not "switch" - without this check, importing/adopting a
	// mirror/aggregate port and applying an unrelated attribute change with
	// op_mode left out of config would silently revert it.
	declaredOpMode := model.OpMode.ValueString()
	explicit := !model.OpMode.IsNull() && !model.OpMode.IsUnknown()
	if explicit && declaredOpMode != "" && declaredOpMode != "switch" {
		po.OpMode = declaredOpMode
	} else if explicit && declaredOpMode == "switch" && current.OpMode != "" && current.OpMode != "switch" {
		po.OpMode = "switch"
	}
	if !model.PoeMode.IsNull() {
		po.PoeMode = model.PoeMode.ValueString()
	}
	if !model.Dot1XCtrl.IsNull() {
		po.Dot1XCtrl = model.Dot1XCtrl.ValueString()
	}
	if !model.FecMode.IsNull() {
		po.FecMode = model.FecMode.ValueString()
	}
	if !model.Forward.IsNull() {
		po.Forward = model.Forward.ValueString()
	}
	if !model.NativeNetworkID.IsNull() {
		po.NATiveNetworkID = model.NativeNetworkID.ValueString()
	}
	if !model.SettingPreference.IsNull() {
		po.SettingPreference = model.SettingPreference.ValueString()
	}
	if !model.StormctrlType.IsNull() {
		po.StormctrlType = model.StormctrlType.ValueString()
	}
	if !model.TaggedVLANMgmt.IsNull() {
		po.TaggedVLANMgmt = model.TaggedVLANMgmt.ValueString()
	}
	if !model.VoiceNetworkID.IsNull() {
		po.VoiceNetworkID = model.VoiceNetworkID.ValueString()
	}

	// These are Optional+Computed with no default and no UseStateForUnknown
	// plan modifier, so a config that leaves them unset plans them Unknown -
	// ValueBool() on an Unknown/Null types.Bool just returns false, which
	// would silently disable the corresponding setting on the controller.
	// Fall back to the port's current controller-side value instead.
	po.Autoneg = boolOrCurrent(model.Autoneg, current.Autoneg)
	po.EgressRateLimitKbpsEnabled = boolOrCurrent(
		model.EgressRateLimitKbpsEnabled,
		current.EgressRateLimitKbpsEnabled,
	)
	po.FlowControlEnabled = boolOrCurrent(model.FlowControlEnabled, current.FlowControlEnabled)
	po.FullDuplex = boolOrCurrent(model.FullDuplex, current.FullDuplex)
	po.Isolation = boolOrCurrent(model.Isolation, current.Isolation)
	po.LldpmedEnabled = boolOrCurrent(model.LldpmedEnabled, current.LldpmedEnabled)
	po.LldpmedNotifyEnabled = boolOrCurrent(
		model.LldpmedNotifyEnabled,
		current.LldpmedNotifyEnabled,
	)
	po.PortKeepaliveEnabled = boolOrCurrent(
		model.PortKeepaliveEnabled,
		current.PortKeepaliveEnabled,
	)
	po.PortSecurityEnabled = boolOrCurrent(model.PortSecurityEnabled, current.PortSecurityEnabled)
	po.StormctrlBroadcastastEnabled = boolOrCurrent(
		model.StormctrlBroadcastEnabled,
		current.StormctrlBroadcastastEnabled,
	)
	po.StormctrlMcastEnabled = boolOrCurrent(
		model.StormctrlMcastEnabled,
		current.StormctrlMcastEnabled,
	)
	po.StormctrlUcastEnabled = boolOrCurrent(
		model.StormctrlUcastEnabled,
		current.StormctrlUcastEnabled,
	)
	po.StpPortMode = boolOrCurrent(model.StpPortMode, current.StpPortMode)

	if !model.Dot1XIDleTimeout.IsNull() {
		po.Dot1XIDleTimeout = util.DurationUnitsPtr(model.Dot1XIDleTimeout, time.Second)
	}
	if !model.EgressRateLimitKbps.IsNull() {
		po.EgressRateLimitKbps = model.EgressRateLimitKbps.ValueInt64Pointer()
	}
	if !model.MirrorPortIDX.IsNull() {
		po.MirrorPortIDX = model.MirrorPortIDX.ValueInt64Pointer()
	}
	if !model.PriorityQueue1Level.IsNull() {
		po.PriorityQueue1Level = model.PriorityQueue1Level.ValueInt64Pointer()
	}
	if !model.PriorityQueue2Level.IsNull() {
		po.PriorityQueue2Level = model.PriorityQueue2Level.ValueInt64Pointer()
	}
	if !model.PriorityQueue3Level.IsNull() {
		po.PriorityQueue3Level = model.PriorityQueue3Level.ValueInt64Pointer()
	}
	if !model.PriorityQueue4Level.IsNull() {
		po.PriorityQueue4Level = model.PriorityQueue4Level.ValueInt64Pointer()
	}
	if !model.Speed.IsNull() {
		po.Speed = model.Speed.ValueInt64Pointer()
	}
	if !model.StormctrlBroadcastLevel.IsNull() {
		po.StormctrlBroadcastastLevel = model.StormctrlBroadcastLevel.ValueInt64Pointer()
	}
	if !model.StormctrlBroadcastRate.IsNull() {
		po.StormctrlBroadcastastRate = model.StormctrlBroadcastRate.ValueInt64Pointer()
	}
	if !model.StormctrlMcastLevel.IsNull() {
		po.StormctrlMcastLevel = model.StormctrlMcastLevel.ValueInt64Pointer()
	}
	if !model.StormctrlMcastRate.IsNull() {
		po.StormctrlMcastRate = model.StormctrlMcastRate.ValueInt64Pointer()
	}
	if !model.StormctrlUcastLevel.IsNull() {
		po.StormctrlUcastLevel = model.StormctrlUcastLevel.ValueInt64Pointer()
	}
	if !model.StormctrlUcastRate.IsNull() {
		po.StormctrlUcastRate = model.StormctrlUcastRate.ValueInt64Pointer()
	}

	if !model.AggregateMembers.IsNull() {
		var aggrMembers []int64
		diags.Append(model.AggregateMembers.ElementsAs(ctx, &aggrMembers, true)...)
		if diags.HasError() {
			return po, diags
		}
		po.AggregateMembers = aggrMembers
	}
	if !model.ExcludedNetworkIDs.IsNull() {
		var excludedIDs []string
		diags.Append(model.ExcludedNetworkIDs.ElementsAs(ctx, &excludedIDs, true)...)
		if diags.HasError() {
			return po, diags
		}
		po.ExcludedNetworkIDs = excludedIDs
	}
	if !model.MulticastRouterNetworkIDs.IsNull() {
		var multicastIDs []string
		diags.Append(model.MulticastRouterNetworkIDs.ElementsAs(ctx, &multicastIDs, true)...)
		if diags.HasError() {
			return po, diags
		}
		po.MulticastRouterNetworkIDs = multicastIDs
	}
	if !model.PortSecurityMACAddress.IsNull() {
		var macAddresses []string
		diags.Append(model.PortSecurityMACAddress.ElementsAs(ctx, &macAddresses, true)...)
		if diags.HasError() {
			return po, diags
		}
		po.PortSecurityMACAddress = macAddresses
	}
	if !model.TaggedNetworkIDs.IsNull() {
		var taggedIDs []string
		diags.Append(model.TaggedNetworkIDs.ElementsAs(ctx, &taggedIDs, true)...)
		if diags.HasError() {
			return po, diags
		}
		po.TaggedNetworkIDs = taggedIDs
	}

	return po, diags
}
