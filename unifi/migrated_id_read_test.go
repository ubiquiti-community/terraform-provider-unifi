package unifi

import (
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

// After the Network 11 upgrade the v2 collections list these objects only
// under their new UUIDs. Read must adopt the object instead of dropping it
// (#716, #742), and must still drop it when it is gone.
func Test_read_adoptsMigratedIDs(t *testing.T) {
	const (
		oldID = "0123456789abcdef01234567"
		newID = "864031f8-6747-498b-9509-7fb704a144a4"
	)
	cases := []struct {
		name  string
		coll  string
		new   func(*Client) crudResource
		state map[string]any
		item  func(id string) map[string]any
	}{
		{
			name: "unifi_dns_record", coll: "static-dns",
			new:   func(c *Client) crudResource { return &dnsRecordFrameworkResource{client: c} },
			state: map[string]any{"name": "*.example.com", "record_type": "A"},
			item: func(id string) map[string]any {
				return map[string]any{
					"_id": id, "key": "*.example.com", "record_type": "A",
					"value": "192.0.2.10", "enabled": true,
				}
			},
		},
		{
			name: "unifi_ap_group", coll: "apgroups",
			new:   func(c *Client) crudResource { return &apGroupResource{client: c} },
			state: map[string]any{"name": "Upstairs"},
			item: func(id string) map[string]any {
				return map[string]any{"_id": id, "name": "Upstairs", "device_macs": []any{}}
			},
		},
		{
			name: "unifi_power_supervisor", coll: "power-supervisors",
			new:   func(c *Client) crudResource { return &powerSupervisorResource{client: c} },
			state: map[string]any{"device_mac": "02:00:00:aa:bb:01"},
			item: func(id string) map[string]any {
				return map[string]any{"id": id, "client_mac": "02:00:00:AA:BB:01", "enabled": true}
			},
		},
		{
			name: "unifi_firewall_zone", coll: "firewall/zone",
			new:   func(c *Client) crudResource { return &firewallZoneResource{client: c} },
			state: map[string]any{"name": "IoT", "zone_key": ""},
			item: func(id string) map[string]any {
				return map[string]any{
					"_id":         id,
					"name":        "IoT",
					"zone_key":    "",
					"network_ids": []any{},
				}
			},
		},
	}
	for _, tc := range cases {
		read := func(t *testing.T, item map[string]any) *fwresource.ReadResponse {
			client, fake := newV2FakeController(t)
			fake.items[tc.coll] = []map[string]any{item}
			r := tc.new(client)
			attrs := map[string]any{"id": oldID, "site": "default"}
			for k, v := range tc.state {
				attrs[k] = v
			}
			state := tfState(t, resourceSchema(t, r), attrs)
			resp := doRead(t, r, state, nil)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Read: %v", resp.Diagnostics)
			}
			return resp
		}
		t.Run(tc.name+"/adopts the re-keyed object", func(t *testing.T) {
			resp := read(t, tc.item(newID))
			if resp.State.Raw.IsNull() {
				t.Fatal("Read dropped an object that still exists under its new ID")
			}
			if got := stateString(t, resp.State, "id"); got != newID {
				t.Errorf("id = %q, want %q", got, newID)
			}
		})
		t.Run(tc.name+"/drops a deleted object", func(t *testing.T) {
			resp := read(t, tc.item("89abcdef0123456789abcdef"))
			if !resp.State.Raw.IsNull() {
				t.Errorf("Read kept an object that is gone: %v", resp.State.Raw)
			}
		})
	}
}
