package unifi

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &deviceResource{}
	_ resource.ResourceWithImportState = &deviceResource{}
)

func NewDeviceFrameworkResource() resource.Resource {
	return &deviceResource{}
}

// deviceResource defines the resource implementation.
type deviceResource struct {
	client *Client
}

// deviceResourceModel describes the resource data model.
type deviceResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Site             types.String `tfsdk:"site"`
	MAC              types.String `tfsdk:"mac"`
	Name             types.String `tfsdk:"name"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	PortOverride     types.Set    `tfsdk:"port_override"`
	AllowAdoption    types.Bool   `tfsdk:"allow_adoption"`
	ForgetOnDestroy  types.Bool   `tfsdk:"forget_on_destroy"`
}

// portOverrideModel describes the port override data model.
type portOverrideModel struct {
	Number            types.Int64  `tfsdk:"number"`
	Name              types.String `tfsdk:"name"`
	PortProfileID     types.String `tfsdk:"port_profile_id"`
	OpMode            types.String `tfsdk:"op_mode"`
	PoeMode           types.String `tfsdk:"poe_mode"`
	AggregateNumPorts types.Int64  `tfsdk:"aggregate_num_ports"`
}

var macAddressRegexp = regexp.MustCompile(`^([a-fA-F0-9]{2}[:-]){5}[a-fA-F0-9]{2}$`)

func (r *deviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *deviceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "`unifi_device` manages a device of the network.\n\n" +
			"Devices are adopted by the controller, so it is not possible for this resource to be created through " +
			"Terraform, the create operation instead will simply start managing the device specified by MAC address. " +
			"It's safer to start this process with an explicit import of the device.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the device.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				Description: "The name of the site to associate the device with.",
				Computed:    true,
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac": schema.StringAttribute{
				Description: "The MAC address of the device. This can be specified so that the provider can take control of a device (since devices are created through adoption).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						macAddressRegexp,
						"Mac address is invalid",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the device.",
				Optional:    true,
				Computed:    true,
			},
			"disabled": schema.BoolAttribute{
				Description: "Specifies whether this device should be disabled.",
				Computed:    true,
			},
			"allow_adoption": schema.BoolAttribute{
				Description: "Specifies whether this resource should tell the controller to adopt the device on create.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"forget_on_destroy": schema.BoolAttribute{
				Description: "Specifies whether this resource should tell the controller to forget the device on destroy.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},

		Blocks: map[string]schema.Block{
			"port_override": schema.SetNestedBlock{
				Description: "Settings overrides for specific switch ports.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"number": schema.Int64Attribute{
							Description: "Switch port number.",
							Required:    true,
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
							Description: "Operating mode of the port, valid values are `switch`, `mirror`, and `aggregate`.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("switch"),
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
						"aggregate_num_ports": schema.Int64Attribute{
							Description: "Number of ports in the aggregate.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(2, 8),
							},
						},
					},
				},
			},
		},
	}
}

func (r *deviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	mac := plan.MAC.ValueString()
	if mac == "" {
		resp.Diagnostics.AddError(
			"MAC Address Required",
			"No MAC address specified, please import the device using terraform import",
		)
		return
	}

	mac = cleanMAC(mac)
	device, err := r.client.GetDeviceByMAC(ctx, site, mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device with MAC %s: %s", mac, err),
		)
		return
	}

	if device == nil {
		resp.Diagnostics.AddError(
			"Device Not Found",
			fmt.Sprintf("Device not found using mac %s", mac),
		)
		return
	}

	if device.Adopted != nil && !*device.Adopted {
		allowAdoption := plan.AllowAdoption.ValueBool()
		if !allowAdoption {
			resp.Diagnostics.AddError(
				"Device Not Adopted",
				"Device must be adopted before it can be managed",
			)
			return
		}

		err := r.client.AdoptDevice(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Adopting Device",
				fmt.Sprintf("Could not adopt device with MAC %s: %s", mac, err),
			)
			return
		}

		device, err = r.waitForDeviceState(
			ctx,
			site, mac,
			unifi.DeviceStateConnected,
			[]unifi.DeviceState{
				unifi.DeviceStateAdopting,
				unifi.DeviceStatePending,
				unifi.DeviceStateProvisioning,
				unifi.DeviceStateUpgrading,
			},
			2*time.Minute,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Waiting for Device Adoption",
				fmt.Sprintf("Could not wait for device adoption: %s", err),
			)
			return
		}
	}

	plan.ID = types.StringValue(device.ID)
	plan.Site = types.StringValue(site)

	// Apply the update operation
	diags = r.updateDevice(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	device, err := r.client.GetDevice(ctx, site, id)
	if err != nil {
		if _, ok := err.(*unifi.NotFoundError); ok {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Device",
			fmt.Sprintf("Could not read device %s: %s", id, err),
		)
		return
	}

	// Update state from API response
	r.setResourceData(ctx, &resp.Diagnostics, device, &state, site)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state deviceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current device state and merge with planned changes
	site := plan.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	id := state.ID.ValueString()
	currentDevice, err := r.client.GetDevice(ctx, site, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Device for Update",
			fmt.Sprintf("Could not read device %s for update: %s", id, err),
		)
		return
	}

	// Apply current API values to state
	r.setResourceData(ctx, &resp.Diagnostics, currentDevice, &state, site)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply plan changes to the state
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		state.Name = plan.Name
	}
	if !plan.PortOverride.IsNull() && !plan.PortOverride.IsUnknown() {
		state.PortOverride = plan.PortOverride
	}

	// Update the resource
	diags = r.updateDevice(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *deviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.ForgetOnDestroy.ValueBool() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	mac := state.MAC.ValueString()
	err := r.client.ForgetDevice(ctx, site, mac)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Forgetting Device",
			fmt.Sprintf("Could not forget device with MAC %s: %s", mac, err),
		)
		return
	}

	_, err = r.waitForDeviceState(
		ctx,
		site, mac,
		unifi.DeviceStatePending,
		[]unifi.DeviceState{unifi.DeviceStateConnected, unifi.DeviceStateDeleting},
		1*time.Minute,
	)
	if _, ok := err.(*unifi.NotFoundError); !ok && err != nil {
		resp.Diagnostics.AddError(
			"Error Waiting for Device Forget",
			fmt.Sprintf("Could not wait for device forget: %s", err),
		)
		return
	}
}

func (r *deviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	site := r.client.Site

	// Handle site:id or site:mac format
	if colons := strings.Count(id, ":"); colons == 1 || colons == 6 {
		importParts := strings.SplitN(id, ":", 2)
		site = importParts[0]
		id = importParts[1]
	}

	// If ID looks like a MAC address, convert to device ID
	if macAddressRegexp.MatchString(id) {
		mac := cleanMAC(id)
		device, err := r.client.GetDeviceByMAC(ctx, site, mac)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Finding Device",
				fmt.Sprintf("Could not find device with MAC %s: %s", mac, err),
			)
			return
		}
		id = device.ID
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
}

// Helper methods

func (r *deviceResource) updateDevice(ctx context.Context, model *deviceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	site := model.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Convert model to API request
	deviceReq, convDiags := r.modelToAPIDevice(ctx, model)
	diags.Append(convDiags...)
	if diags.HasError() {
		return diags
	}

	deviceReq.ID = model.ID.ValueString()
	deviceReq.SiteID = site

	device, err := r.client.UpdateDevice(ctx, site, deviceReq)
	if err != nil {
		diags.AddError(
			"Error Updating Device",
			fmt.Sprintf("Could not update device: %s", err),
		)
		return diags
	}

	// Wait for device to be in connected state
	device, err = r.waitForDeviceState(
		ctx,
		site, device.MAC,
		unifi.DeviceStateConnected,
		[]unifi.DeviceState{unifi.DeviceStateAdopting, unifi.DeviceStateProvisioning},
		1*time.Minute,
	)
	if err != nil {
		diags.AddError(
			"Error Waiting for Device Update",
			fmt.Sprintf("Could not wait for device update: %s", err),
		)
		return diags
	}

	// Update state from API response
	r.setResourceData(ctx, &diags, device, model, site)
	return diags
}

func (r *deviceResource) setResourceData(ctx context.Context, diags *diag.Diagnostics, device *unifi.Device, model *deviceResourceModel, site string) {
	model.Site = types.StringValue(site)

	if device.MAC == "" {
		model.MAC = types.StringNull()
	} else {
		model.MAC = types.StringValue(device.MAC)
	}

	if device.Name == "" {
		model.Name = types.StringNull()
	} else {
		model.Name = types.StringValue(device.Name)
	}

	if device.Disabled != nil {
		model.Disabled = types.BoolValue(*device.Disabled)
	} else {
		model.Disabled = types.BoolValue(false)
	}

	// Convert port overrides
	portOverrides, convDiags := r.portOverridesToFramework(ctx, device.PortOverrides)
	diags.Append(convDiags...)
	if !diags.HasError() {
		model.PortOverride = portOverrides
	}
}

func (r *deviceResource) modelToAPIDevice(ctx context.Context, model *deviceResourceModel) (*unifi.Device, diag.Diagnostics) {
	var diags diag.Diagnostics

	device := &unifi.Device{
		MAC:  model.MAC.ValueString(),
		Name: model.Name.ValueString(),
	}

	// Convert port overrides
	if !model.PortOverride.IsNull() && !model.PortOverride.IsUnknown() {
		portOverrides, convDiags := r.frameworkToPortOverrides(ctx, model.PortOverride)
		diags.Append(convDiags...)
		if !diags.HasError() {
			device.PortOverrides = portOverrides
		}
	}

	return device, diags
}

func (r *deviceResource) portOverridesToFramework(ctx context.Context, pos []unifi.DevicePortOverrides) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(pos) == 0 {
		return types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"number":              types.Int64Type,
				"name":                types.StringType,
				"port_profile_id":     types.StringType,
				"op_mode":             types.StringType,
				"poe_mode":            types.StringType,
				"aggregate_num_ports": types.Int64Type,
			},
		}), diags
	}

	attrType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"number":              types.Int64Type,
			"name":                types.StringType,
			"port_profile_id":     types.StringType,
			"op_mode":             types.StringType,
			"poe_mode":            types.StringType,
			"aggregate_num_ports": types.Int64Type,
		},
	}

	elements := make([]attr.Value, 0, len(pos))
	for _, po := range pos {
		portOverrideObj := map[string]attr.Value{
			"number": types.Int64Value(int64(po.PortIDX)),
		}

		if po.Name == "" {
			portOverrideObj["name"] = types.StringNull()
		} else {
			portOverrideObj["name"] = types.StringValue(po.Name)
		}

		if po.PortProfileID == "" {
			portOverrideObj["port_profile_id"] = types.StringNull()
		} else {
			portOverrideObj["port_profile_id"] = types.StringValue(po.PortProfileID)
		}

		if po.OpMode == "" {
			portOverrideObj["op_mode"] = types.StringValue("switch")
		} else {
			portOverrideObj["op_mode"] = types.StringValue(po.OpMode)
		}

		if po.PoeMode == "" {
			portOverrideObj["poe_mode"] = types.StringNull()
		} else {
			portOverrideObj["poe_mode"] = types.StringValue(po.PoeMode)
		}

		if po.AggregateNumPorts == 0 {
			portOverrideObj["aggregate_num_ports"] = types.Int64Null()
		} else {
			portOverrideObj["aggregate_num_ports"] = types.Int64Value(int64(po.AggregateNumPorts))
		}

		objVal, objDiags := types.ObjectValue(attrType.AttrTypes, portOverrideObj)
		diags.Append(objDiags...)
		if diags.HasError() {
			continue
		}
		elements = append(elements, objVal)
	}

	if diags.HasError() {
		return types.SetNull(attrType), diags
	}

	setValue, setDiags := types.SetValue(attrType, elements)
	diags.Append(setDiags...)
	return setValue, diags
}

func (r *deviceResource) frameworkToPortOverrides(ctx context.Context, portOverrideSet types.Set) ([]unifi.DevicePortOverrides, diag.Diagnostics) {
	var diags diag.Diagnostics

	elements := portOverrideSet.Elements()
	overrideMap := make(map[int]unifi.DevicePortOverrides)

	for _, elem := range elements {
		obj := elem.(types.Object)
		attrs := obj.Attributes()

		idx := int(attrs["number"].(types.Int64).ValueInt64())

		po := unifi.DevicePortOverrides{
			PortIDX: idx,
		}

		if nameAttr, ok := attrs["name"]; ok && !nameAttr.IsNull() {
			po.Name = nameAttr.(types.String).ValueString()
		}

		if profileAttr, ok := attrs["port_profile_id"]; ok && !profileAttr.IsNull() {
			po.PortProfileID = profileAttr.(types.String).ValueString()
		}

		if opModeAttr, ok := attrs["op_mode"]; ok && !opModeAttr.IsNull() {
			po.OpMode = opModeAttr.(types.String).ValueString()
		}

		if poeModeAttr, ok := attrs["poe_mode"]; ok && !poeModeAttr.IsNull() {
			po.PoeMode = poeModeAttr.(types.String).ValueString()
		}

		if aggAttr, ok := attrs["aggregate_num_ports"]; ok && !aggAttr.IsNull() {
			po.AggregateNumPorts = int(aggAttr.(types.Int64).ValueInt64())
		}

		overrideMap[idx] = po
	}

	pos := make([]unifi.DevicePortOverrides, 0, len(overrideMap))
	for _, po := range overrideMap {
		pos = append(pos, po)
	}

	return pos, diags
}

func (r *deviceResource) waitForDeviceState(
	ctx context.Context,
	site, mac string,
	targetState unifi.DeviceState,
	pendingStates []unifi.DeviceState,
	timeout time.Duration,
) (*unifi.Device, error) {
	// Always consider unknown to be a pending state.
	pendingStates = append(pendingStates, unifi.DeviceStateUnknown)

	var pending []string
	for _, state := range pendingStates {
		pending = append(pending, state.String())
	}

	wait := retry.StateChangeConf{
		Pending: pending,
		Target:  []string{targetState.String()},
		Refresh: func() (any, string, error) {
			device, err := r.client.GetDeviceByMAC(ctx, site, mac)

			if _, ok := err.(*unifi.NotFoundError); ok {
				err = nil
			}

			// When a device is forgotten, it will disappear from the UI for a few seconds before reappearing.
			// During this time, `device.GetDeviceByMAC` will return a 400.
			if err != nil && strings.Contains(err.Error(), "api.err.UnknownDevice") {
				err = nil
			}

			var state string
			if device != nil {
				state = device.State.String()
			}

			if device == nil {
				return nil, state, err
			}

			return device, state, err
		},
		Timeout:        timeout,
		NotFoundChecks: 30,
	}

	outputRaw, err := wait.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*unifi.Device); ok {
		return output, err
	}

	return nil, err
}

// cleanMAC normalizes MAC address format
func cleanMAC(mac string) string {
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ToLower(mac)
	return mac
}