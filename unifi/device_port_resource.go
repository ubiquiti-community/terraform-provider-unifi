package unifi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// devicePortResource manages the overrides of one switch port as a resource
// of its own, keyed by device MAC and port index, so that a single port can
// be imported and managed without declaring the whole device. The controller
// has no per-port endpoint: every write reads the device, replaces this port's
// entry in its port_overrides array and writes the device back, exactly as the
// port_override block of unifi_device does for the ports it declares.
var (
	_ resource.Resource                = &devicePortResource{}
	_ resource.ResourceWithConfigure   = &devicePortResource{}
	_ resource.ResourceWithImportState = &devicePortResource{}
	_ resource.ResourceWithIdentity    = &devicePortResource{}
)

func NewDevicePortResource() resource.Resource {
	return &devicePortResource{}
}

type devicePortResource struct {
	client *Client
}

// devicePortResourceModel is one port override (the same attributes as a
// port_override block of unifi_device) plus the keys that locate it.
type devicePortResourceModel struct {
	portOverrideModel

	ID        types.String       `tfsdk:"id"`
	Site      types.String       `tfsdk:"site"`
	DeviceMAC hwtypes.MACAddress `tfsdk:"device_mac"`
	Timeouts  timeouts.Value     `tfsdk:"timeouts"`
}

type devicePortIdentityModel struct {
	DeviceMAC hwtypes.MACAddress `tfsdk:"device_mac"`
	Index     types.Int64        `tfsdk:"index"`
	Site      types.String       `tfsdk:"site"`
}

func devicePortIdentity(mac string, index int64, site string) devicePortIdentityModel {
	return devicePortIdentityModel{
		DeviceMAC: hwtypes.NewMACAddressValue(mac),
		Index:     types.Int64Value(index),
		Site:      types.StringValue(site),
	}
}

// devicePortID is the `id` attribute and the import ID: "<device_mac>/<index>".
func devicePortID(mac string, index int64) string {
	return fmt.Sprintf("%s/%d", mac, index)
}

func (r *devicePortResource) Metadata(
	_ context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_device_port"
}

func (r *devicePortResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"device_mac": identityschema.StringAttribute{
				CustomType:        hwtypes.MACAddressType{},
				RequiredForImport: true,
			},
			"index": identityschema.Int64Attribute{RequiredForImport: true},
			"site":  identityschema.StringAttribute{OptionalForImport: true},
		},
	}
}

func (r *devicePortResource) Schema(
	ctx context.Context,
	_ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	attrs := devicePortOverrideAttributes()
	attrs["id"] = schema.StringAttribute{
		Description: "`<device_mac>/<index>`.",
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	attrs["site"] = schema.StringAttribute{
		Description: "The name of the site the device belongs to. Defaults to the provider's site.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
			stringplanmodifier.RequiresReplaceIfConfigured(),
		},
	}
	attrs["device_mac"] = schema.StringAttribute{
		Description: "MAC address of the switch the port belongs to.",
		CustomType:  hwtypes.MACAddressType{},
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	attrs["index"] = schema.Int64Attribute{
		Description: "Switch port index, starting at 1.",
		Required:    true,
		Validators: []validator.Int64{
			int64validator.AtLeast(1),
		},
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	}
	attrs["timeouts"] = timeouts.Attributes(
		ctx,
		timeouts.Opts{Create: true, Read: true, Update: true, Delete: true},
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the overrides of one switch port, identified by `device_mac` and " +
			"`index`, with the same settings as a `port_override` block of `unifi_device`. " +
			"Use it to manage a port without declaring the whole device, and to import a single " +
			"port: the import ID is `<device_mac>/<index>` (or `<site>/<device_mac>/<index>`), " +
			"which a `port_override` block has no equivalent of. An import stores every setting " +
			"the controller holds for the port, so the first plan shows what the configuration " +
			"does not declare; applying it clears those settings, as for a `port_override` block.\n\n" +
			"The controller has no per-port endpoint, so a write reads the device, replaces this " +
			"port's entry in its `port_overrides` array and writes the device back; the other " +
			"ports are left as the controller has them. Attributes left unset keep the " +
			"controller's value. Writers of the same device from one provider process are " +
			"serialised, so several `unifi_device_port` resources of one switch can be applied " +
			"in parallel.\n\n" +
			"Do not declare the same port both here and in a `port_override` block of " +
			"`unifi_device`: the two write the same controller field, the last write wins and " +
			"the other resource drifts. Destroying the resource stops managing the port and " +
			"leaves its current settings in place, like removing a `port_override` block; reset " +
			"the port to its defaults before destroying it if that is what you want.",
		Attributes: attrs,
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

	timeout, d := plan.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	state, site, diags := r.write(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, devicePortIdentity(
		cleanMAC(state.DeviceMAC.ValueString()), state.Index.ValueInt64(), site,
	))...)
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

	timeout, d := plan.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	state, _, diags := r.write(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// write sends the planned port override to the controller and returns the
// state to store: the plan reconciled with the entry the controller holds
// after the write, and the site that was used.
func (r *devicePortResource) write(
	ctx context.Context,
	plan devicePortResourceModel,
) (devicePortResourceModel, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	mac := cleanMAC(plan.DeviceMAC.ValueString())
	index := plan.Index.ValueInt64()
	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	unlock := r.client.lockDevice(deviceLockKey(mac, ""))
	defer unlock()

	device, err := r.client.GetDeviceByMAC(ctx, site, mac)
	var notFound *unifi.NotFoundError
	switch {
	case errors.As(err, &notFound), err == nil && device == nil:
		diags.AddError(
			"Device Not Found",
			fmt.Sprintf("No device with MAC %s on site %s.", mac, site),
		)
		return plan, site, diags
	case err != nil:
		diags.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device %s: %s", mac, err),
		)
		return plan, site, diags
	}

	if n, ok := devicePortExists(device.PortTable, index); !ok {
		diags.AddError(
			"Port Not Found",
			fmt.Sprintf(
				"Device %s has no port %d: its port table lists %d ports.",
				mac, index, n,
			),
		)
		return plan, site, diags
	}

	// A port can be managed before it has an override entry: the entry is then
	// appended, as for a new port_override block.
	declared, d := portOverrideModelToAPI(ctx, plan.portOverrideModel)
	diags.Append(d...)
	if diags.HasError() {
		return plan, site, diags
	}

	merged := mergePortOverridesByIndex(
		device.PortOverrides,
		[]unifi.DevicePortOverrides{declared},
	)
	body := buildMinimalUpdateDevice(device, device, merged)
	if _, err := r.client.UpdateDevice(ctx, site, body); err != nil {
		diags.AddError(
			"Error Updating Device Port",
			fmt.Sprintf("Could not update port %d of device %s: %s", index, mac, err),
		)
		return plan, site, diags
	}

	// The PUT provisions asynchronously: read the device back once it has
	// settled, so the state reflects what the controller kept.
	updated, err := waitForDeviceState(
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
				mac, index, err,
			),
		)
		return plan, site, diags
	}

	apiPO, found := findPortOverride(updated.PortOverrides, index)
	if !found {
		diags.AddError(
			"Port Missing After Update",
			fmt.Sprintf(
				"Device %s reports no override for port %d after the update.",
				mac, index,
			),
		)
		return plan, site, diags
	}

	state := plan
	state.ID = types.StringValue(devicePortID(mac, index))
	state.Site = types.StringValue(site)
	state.portOverrideModel, d = reconcilePortOverride(plan.portOverrideModel, apiPO)
	diags.Append(d...)
	return state, site, diags
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

	timeout, d := state.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// An identity-based import stores only the identity; take the keys from it
	// when the state has none.
	var identity devicePortIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if state.DeviceMAC.IsNull() || state.DeviceMAC.IsUnknown() {
		state.DeviceMAC = identity.DeviceMAC
	}
	if state.Index.IsNull() || state.Index.IsUnknown() {
		state.Index = identity.Index
	}
	site := state.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}
	mac := cleanMAC(state.DeviceMAC.ValueString())
	index := state.Index.ValueInt64()
	if mac == "" || index == 0 {
		resp.Diagnostics.AddError(
			"Incomplete Device Port State",
			"The state carries neither device_mac and index nor a resource identity.",
		)
		return
	}
	id := devicePortIdentity(mac, index, site)

	// Read under the device's lock so a write in progress by another resource
	// of the same device does not show a half-provisioned port.
	unlock := r.client.lockDevice(deviceLockKey(mac, ""))
	defer unlock()

	device, err := r.client.GetDeviceByMAC(ctx, site, mac)
	if err != nil {
		var notFound *unifi.NotFoundError
		if errors.As(err, &notFound) {
			removeResourceOnRead(ctx, resp, id)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device %s: %s", mac, err),
		)
		return
	}
	if device == nil {
		removeResourceOnRead(ctx, resp, id)
		return
	}

	apiPO, found := findPortOverride(device.PortOverrides, index)
	switch {
	case state.ID.IsNull() || state.ID.ValueString() == "":
		// First read after an import: the state holds only the keys, so take
		// every attribute from the controller. A port without an override entry
		// has nothing to import; it can only be created.
		if !found {
			resp.Diagnostics.AddError(
				"Port Has No Overrides",
				fmt.Sprintf(
					"Device %s has no override entry for port %d, so there is nothing to "+
						"import. Declare the port in configuration and apply it instead.",
					mac, index,
				),
			)
			return
		}
		state.portOverrideModel, d = apiPortOverrideToModel(apiPO)
		// op_mode defaults to "switch" in the schema, and the controller leaves
		// it out for a port in that mode: store the default so the first plan
		// after the import does not set it.
		if state.OpMode.IsNull() {
			state.OpMode = types.StringValue("switch")
		}
	case found:
		state.portOverrideModel, d = reconcilePortOverride(state.portOverrideModel, apiPO)
	default:
		// The port lost its override entry (cleared on the controller): every
		// attribute the configuration sets now reads as unset, so the plan
		// shows the drift and the next apply writes the entry again.
		state.portOverrideModel, d = reconcilePortOverride(
			state.portOverrideModel,
			unifi.DevicePortOverrides{PortIDX: &index},
		)
	}
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(devicePortID(mac, index))
	state.Site = types.StringValue(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if req.Identity == nil || req.Identity.Raw.IsNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, id)...)
	}
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

	// Same semantics as removing a port_override block: the port is no longer
	// managed, its current settings stay on the controller.
	resp.Diagnostics.AddWarning(
		"Port Settings Left in Place",
		fmt.Sprintf(
			"Port %d of device %s is no longer managed; its current settings were left on "+
				"the controller.",
			state.Index.ValueInt64(), state.DeviceMAC.ValueString(),
		),
	)
}

func (r *devicePortResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	var (
		mac   string
		index int64
		site  = r.client.Site
	)
	if req.ID == "" {
		// Import by resource identity (import block with identity, Terraform 1.12+).
		var identity devicePortIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		mac = cleanMAC(identity.DeviceMAC.ValueString())
		index = identity.Index.ValueInt64()
		if s := identity.Site.ValueString(); s != "" {
			site = s
		}
		if mac == "" || index < 1 {
			resp.Diagnostics.AddError(
				"Invalid Import Identity",
				"The identity must carry device_mac and an index of 1 or more.",
			)
			return
		}
	} else {
		var err error
		var idSite string
		idSite, mac, index, err = parseDevicePortImportID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", err.Error())
			return
		}
		if idSite != "" {
			site = idSite
		}
	}

	// Only the keys are stored here; the following Read fills in every other
	// attribute from the controller (it recognises the import by the missing id).
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx, path.Root("device_mac"), hwtypes.NewMACAddressValue(mac))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("index"), index)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, devicePortIdentity(mac, index, site))...)
}

// parseDevicePortImportID splits "<device_mac>/<index>" or
// "<site>/<device_mac>/<index>". The separator is a slash because the MAC
// address itself is colon-separated. The MAC is normalised with cleanMAC.
func parseDevicePortImportID(importID string) (site, mac string, index int64, err error) {
	parts := strings.Split(importID, "/")
	switch len(parts) {
	case 2:
		mac = parts[0]
	case 3:
		site, mac = parts[0], parts[1]
	default:
		return "", "", 0, fmt.Errorf(
			`import ID must be "<device_mac>/<index>" or "<site>/<device_mac>/<index>", got %q`,
			importID,
		)
	}
	if len(parts) == 3 && site == "" {
		return "", "", 0, fmt.Errorf("import ID %q has an empty site", importID)
	}

	mac = cleanMAC(mac)
	if hw, parseErr := net.ParseMAC(mac); parseErr != nil || len(hw) != 6 {
		return "", "", 0, fmt.Errorf("%q is not a MAC address", mac)
	}

	index, err = strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return "", "", 0, fmt.Errorf(
			"could not parse port index %q as an integer: %w",
			parts[len(parts)-1], err,
		)
	}
	if index < 1 {
		return "", "", 0, fmt.Errorf("port index must be 1 or more, got %d", index)
	}

	return site, mac, index, nil
}

// findPortOverride returns the override entry of port index, if the device has one.
// devicePortExists reports whether the device's port table lists the port, and
// how many ports the table has. An empty table (device not yet provisioned, or
// a controller that does not report it) cannot be checked and counts as a match,
// the controller then rejects an index the device does not have.
func devicePortExists(table []unifi.DevicePortTable, index int64) (int, bool) {
	if len(table) == 0 {
		return 0, true
	}
	for _, p := range table {
		if p.PortIdx == index {
			return len(table), true
		}
	}
	return len(table), false
}

func findPortOverride(
	overrides []unifi.DevicePortOverrides,
	index int64,
) (unifi.DevicePortOverrides, bool) {
	for _, po := range overrides {
		if po.PortIDX != nil && *po.PortIDX == index {
			return po, true
		}
	}
	return unifi.DevicePortOverrides{}, false
}
