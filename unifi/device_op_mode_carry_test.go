package unifi

import (
	"reflect"
	"testing"

	"github.com/ubiquiti-community/go-unifi/unifi"
)

// A declared port_override must not delete an op_mode the controller already has.
//
// modelToAPIPortOverride never writes op_mode when it is "switch", so a declared
// entry for such a port arrives here with OpMode == "". Substituting it wholesale
// drops the field from the marshalled entry, which makes the merged array differ
// from the controller's. UpdateDevice PUTs getDeviceDiff(existing, target) and
// compares port_overrides as one JSON value, so that single difference sends the
// whole array — and the array has been round-tripped through DevicePortOverrides,
// which drops unmodelled fields and every zero value. The result is that declaring
// one port strips settings from every port on the device.
func Test_mergePortOverridesByIndex_carriesOpModeForward(t *testing.T) {
	tests := []struct {
		name     string
		current  []unifi.DevicePortOverrides
		declared []unifi.DevicePortOverrides
		want     []unifi.DevicePortOverrides
	}{
		{
			name: "declared entry without op_mode keeps the controller's value",
			current: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(7), Name: "Port 7", OpMode: "switch"},
			},
			declared: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(7), Name: "Port 7 renamed"},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(7), Name: "Port 7 renamed", OpMode: "switch"},
			},
		},
		{
			name: "a declared non-switch op_mode still wins",
			current: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(9), OpMode: "switch"},
			},
			declared: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(9), OpMode: "aggregate"},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(9), OpMode: "aggregate"},
			},
		},
		{
			name: "nothing is invented when the controller has no op_mode",
			current: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(5), Name: "Dell-01"},
			},
			declared: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(5), Name: "Dell-01", PoeMode: "auto"},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(5), Name: "Dell-01", PoeMode: "auto"},
			},
		},
		{
			name: "undeclared ports are untouched",
			current: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(1), Name: "Uplink", OpMode: "switch"},
				{PortIDX: ptrInt64(7), Name: "Port 7", OpMode: "switch"},
			},
			declared: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(7), Name: "Port 7 renamed"},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(1), Name: "Uplink", OpMode: "switch"},
				{PortIDX: ptrInt64(7), Name: "Port 7 renamed", OpMode: "switch"},
			},
		},
		{
			name: "a newly-managed port with no current entry is added as declared",
			current: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(1), OpMode: "switch"},
			},
			declared: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(2), Name: "new"},
			},
			want: []unifi.DevicePortOverrides{
				{PortIDX: ptrInt64(1), OpMode: "switch"},
				{PortIDX: ptrInt64(2), Name: "new"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergePortOverridesByIndex(tt.current, tt.declared); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergePortOverridesByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
