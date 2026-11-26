package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A state written by schema version 0 can hold `false` for every flag of a
// declared port. Upgraded as they are, the flags that became nested groups are
// objects where the configuration has null and the rest are values the block
// does not set, so no element lines up with its block any more.
func TestDeviceUpgradeState_v0DropsUnsetPortFlags(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	// Port 1 as a version 0 provider stored the lead port of a LAG, port 2
	// with flags that are set.
	prior := []byte(`{
		"id": "dev-1", "site": "default", "mac": "00:11:22:33:44:55", "name": "USW Aggregation",
		"port_override": [{
			"index": 1, "name": "node-a LACP", "op_mode": "aggregate", "forward": "customize",
			"aggregate_members": [1, 2],
			"native_networkconf_id": "0a1b2c3d-1111-4222-8333-444455556666",
			"autoneg": false, "flow_control_enabled": false, "full_duplex": false,
			"isolation": false, "port_keepalive_enabled": false, "stp_port_mode": false,
			"egress_rate_limit_kbps_enabled": false,
			"lldpmed_enabled": false, "lldpmed_notify_enabled": false,
			"port_security_enabled": false,
			"stormctrl_bcast_enabled": false, "stormctrl_mcast_enabled": false,
			"stormctrl_ucast_enabled": false
		}, {
			"index": 2, "name": "camera", "op_mode": "switch",
			"autoneg": true, "isolation": false,
			"lldpmed_enabled": true, "lldpmed_notify_enabled": false
		}]
	}`)

	resp := &fwresource.UpgradeStateResponse{}
	r.UpgradeState(ctx)[0].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: prior},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade failed: %v", resp.Diagnostics)
	}
	val, err := resp.DynamicValue.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal upgraded value: %v", err)
	}
	var root map[string]tftypes.Value
	if err := val.As(&root); err != nil {
		t.Fatalf("as object: %v", err)
	}
	var overrides []tftypes.Value
	if err := root["port_override"].As(&overrides); err != nil || len(overrides) != 2 {
		t.Fatalf("port_override = %v (%v), want two elements", root["port_override"], err)
	}
	ports := map[string]map[string]tftypes.Value{}
	for _, o := range overrides {
		var po map[string]tftypes.Value
		if err := o.As(&po); err != nil {
			t.Fatalf("port_override element: %v", err)
		}
		var name string
		if err := po["name"].As(&name); err != nil {
			t.Fatalf("name: %v", err)
		}
		ports[name] = po
	}

	lag := ports["node-a LACP"]
	for name, v := range lag {
		switch name {
		case "index", "name", "op_mode", "forward", "aggregate_members",
			"native_networkconf_id":
			if v.IsNull() {
				t.Errorf("LAG port: %s is null, want the stored value", name)
			}
		default:
			if !v.IsNull() {
				t.Errorf("LAG port: %s = %v, want null", name, v)
			}
		}
	}

	cam := ports["camera"]
	var autoneg bool
	if err := cam["autoneg"].As(&autoneg); err != nil || !autoneg {
		t.Errorf("camera: autoneg = %v (%v), want true", cam["autoneg"], err)
	}
	if !cam["isolation"].IsNull() {
		t.Errorf("camera: isolation = %v, want null", cam["isolation"])
	}
	var lldpmed map[string]tftypes.Value
	if err := cam["lldpmed"].As(&lldpmed); err != nil {
		t.Fatalf("camera: lldpmed = %v (%v), want an object", cam["lldpmed"], err)
	}
	var enabled bool
	if err := lldpmed["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf("camera: lldpmed.enabled = %v (%v), want true", lldpmed["enabled"], err)
	}
	if !lldpmed["notify_enabled"].IsNull() {
		t.Errorf("camera: lldpmed.notify_enabled = %v, want null", lldpmed["notify_enabled"])
	}
}
