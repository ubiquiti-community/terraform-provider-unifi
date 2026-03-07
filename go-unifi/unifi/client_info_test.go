package unifi

import (
	"encoding/json"
	"testing"
)

func TestClientInfoDeserialization(t *testing.T) {
	tests := []struct {
		name          string
		raw           string
		wantPort      *int64
		wantUplinkMAC string
		wantDisplay   string
	}{
		{
			name:          "direct object",
			raw:           `{"display_name":"talos","mac":"2c:cf:67:0a:a3:33","last_uplink_mac":"f4:e2:c6:50:60:bb","last_uplink_remote_port":5,"status":"DISCONNECTED"}`,
			wantPort:      ptrInt64(5),
			wantUplinkMAC: "f4:e2:c6:50:60:bb",
			wantDisplay:   "talos",
		},
		{
			name:          "meta/data array wrapper silently fails",
			raw:           `{"meta":{"rc":"ok"},"data":[{"display_name":"talos","mac":"2c:cf:67:0a:a3:33","last_uplink_mac":"f4:e2:c6:50:60:bb","last_uplink_remote_port":5}]}`,
			wantPort:      nil,
			wantUplinkMAC: "",
			wantDisplay:   "",
		},
		{
			name:          "data object wrapper silently fails",
			raw:           `{"data":{"display_name":"talos","mac":"2c:cf:67:0a:a3:33","last_uplink_mac":"f4:e2:c6:50:60:bb","last_uplink_remote_port":5}}`,
			wantPort:      nil,
			wantUplinkMAC: "",
			wantDisplay:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ci ClientInfo
			err := json.Unmarshal([]byte(tt.raw), &ci)
			if err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}
			t.Logf("DisplayName=%q LastUplinkMac=%q LastUplinkRemotePort=%v SwPort=%v Status=%q",
				ci.DisplayName, ci.LastUplinkMac, ci.LastUplinkRemotePort, ci.SwPort, ci.Status)

			if tt.wantPort == nil {
				if ci.LastUplinkRemotePort != nil {
					t.Errorf("expected nil port, got %d", *ci.LastUplinkRemotePort)
				}
			} else {
				if ci.LastUplinkRemotePort == nil {
					t.Errorf("expected port %d, got nil", *tt.wantPort)
				} else if *ci.LastUplinkRemotePort != *tt.wantPort {
					t.Errorf("expected port %d, got %d", *tt.wantPort, *ci.LastUplinkRemotePort)
				}
			}

			if ci.LastUplinkMac != tt.wantUplinkMAC {
				t.Errorf("expected uplink MAC %q, got %q", tt.wantUplinkMAC, ci.LastUplinkMac)
			}
			if ci.DisplayName != tt.wantDisplay {
				t.Errorf("expected display %q, got %q", tt.wantDisplay, ci.DisplayName)
			}
		})
	}
}

func ptrInt64(v int64) *int64 { return &v }
