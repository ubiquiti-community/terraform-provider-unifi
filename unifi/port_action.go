package unifi

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/action/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure the implementation satisfies framework interfaces.
var (
	_ action.Action              = &portAction{}
	_ action.ActionWithConfigure = &portAction{}
)

// NewPortAction returns a new instance of the port action.
func NewPortAction() action.Action {
	return &portAction{}
}

type portAction struct {
	client *Client
}

// portActionModel describes the action request and response data model.
type portActionModel struct {
	DeviceMAC  hwtypes.MACAddress `tfsdk:"device_mac"`
	PortNumber types.Int64        `tfsdk:"port_number"`
	PoeMode    types.String       `tfsdk:"poe_mode"`
	Timeouts   timeouts.Value     `tfsdk:"timeouts"`
}

func (a *portAction) Metadata(
	ctx context.Context,
	req action.MetadataRequest,
	resp *action.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_port"
}

func (a *portAction) Schema(
	ctx context.Context,
	req action.SchemaRequest,
	resp *action.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Performs an action on a UniFi device port, allowing configuration of PoE state and other port settings.",

		Attributes: map[string]schema.Attribute{
			"device_mac": schema.StringAttribute{
				MarkdownDescription: "MAC address of the device containing the port to configure.",
				CustomType:          hwtypes.MACAddressType{},
				Required:            true,
			},
			"port_number": schema.Int64Attribute{
				MarkdownDescription: "Port number (index) on the device to configure. Typically starts at 1.",
				Required:            true,
			},
			"poe_mode": schema.StringAttribute{
				MarkdownDescription: "PoE mode to set for the port. Valid values are `auto`, `pasv24`, `passthrough`, and `off`.",
				Required:            true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

func (a *portAction) Configure(
	ctx context.Context,
	req action.ConfigureRequest,
	resp *action.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Action Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	a.client = client
}

func (a *portAction) Invoke(
	ctx context.Context,
	req action.InvokeRequest,
	resp *action.InvokeResponse,
) {
	var config portActionModel

	// Read the action configuration
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	invokeTimeout, timeoutDiags := config.Timeouts.Invoke(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, invokeTimeout)
	defer cancel()

	// Validate PoE mode
	poeMode := config.PoeMode.ValueString()
	validPoeModes := []string{"auto", "pasv24", "passthrough", "off"}
	isValidPoeMode := slices.Contains(validPoeModes, poeMode)
	if !isValidPoeMode {
		resp.Diagnostics.AddError(
			"Invalid PoE Mode",
			fmt.Sprintf(
				"PoE mode must be one of: auto, pasv24, passthrough, off. Got: %s",
				poeMode,
			),
		)
		return
	}

	deviceMAC := config.DeviceMAC.ValueString()
	portNumber := config.PortNumber.ValueInt64()

	// This does the same unlocked read-device/modify-port_overrides/PUT-device
	// sequence as unifi_device_port and unifi_device's port_override, so it
	// takes the same per-device lock (keyed the same way, cleanMAC(mac)) to
	// avoid racing either of them and silently dropping one side's change.
	unlock := a.client.lockDevice(cleanMAC(deviceMAC))
	defer unlock()

	// Get the device first to retrieve its ID
	device, err := a.client.GetDeviceByMAC(ctx, a.client.Site, deviceMAC)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Finding Device",
			fmt.Sprintf("Could not find device with MAC address %s: %s", deviceMAC, err.Error()),
		)
		return
	}

	// Update the device with port override for PoE configuration
	portOverride := ui.DevicePortOverrides{
		PortIDX: config.PortNumber.ValueInt64Pointer(),
		PoeMode: poeMode,
	}

	// Check if the device already has port overrides
	existingOverrides := device.PortOverrides
	if existingOverrides == nil {
		existingOverrides = []ui.DevicePortOverrides{}
	}

	// Find and update existing override or add new one
	found := false
	for i, override := range existingOverrides {
		if override.PortIDX == config.PortNumber.ValueInt64Pointer() {
			// Update existing override
			existingOverrides[i].PoeMode = poeMode
			found = true
			break
		}
	}

	if !found {
		// Add new override
		existingOverrides = append(existingOverrides, portOverride)
	}

	// Update the device
	device.PortOverrides = existingOverrides
	_, err = a.client.UpdateDevice(ctx, a.client.Site, device)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Device Port",
			fmt.Sprintf(
				"Could not update port %d on device %s: %s",
				portNumber,
				deviceMAC,
				err.Error(),
			),
		)
		return
	}

	// UpdateDevice's PUT provisions asynchronously. Wait for the device to
	// settle back to connected before returning (and releasing the deferred
	// lock above) so a concurrent unifi_device_port/unifi_device writer that
	// then acquires the lock can't read pre-update port_overrides.
	if _, err := waitForDeviceState(
		ctx,
		a.client,
		a.client.Site, deviceMAC,
		ui.DeviceStateConnected,
		[]ui.DeviceState{ui.DeviceStateAdopting, ui.DeviceStateProvisioning},
		3*time.Minute,
	); err != nil {
		resp.Diagnostics.AddError(
			"Error Waiting for Device Port Update",
			fmt.Sprintf(
				"Could not wait for device %s to settle after updating port %d: %s",
				deviceMAC,
				portNumber,
				err,
			),
		)
		return
	}

	// Action completed successfully - no result data to return for actions
}
