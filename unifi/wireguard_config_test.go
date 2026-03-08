package unifi

import (
	"encoding/base64"
	"testing"
)

func TestParseWireGuardConfig(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		checkResult func(t *testing.T, result *wireguardConfigParsed)
	}{
		{
			name: "full_config",
			content: `[Interface]
PrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=
Address = 10.0.0.2/24

[Peer]
PublicKey = 7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=
Endpoint = 192.0.2.1:51820
AllowedIPs = 0.0.0.0/0
`,
			checkResult: func(t *testing.T, r *wireguardConfigParsed) {
				if r.PrivateKey != "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=" {
					t.Errorf("PrivateKey = %q", r.PrivateKey)
				}
				if r.Address != "10.0.0.2/24" {
					t.Errorf("Address = %q", r.Address)
				}
				if r.PublicKey != "7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=" {
					t.Errorf("PublicKey = %q", r.PublicKey)
				}
				if r.EndpointIP != "192.0.2.1" {
					t.Errorf("EndpointIP = %q", r.EndpointIP)
				}
				if r.EndpointPort != 51820 {
					t.Errorf("EndpointPort = %d", r.EndpointPort)
				}
				if r.AllowedIPs != "0.0.0.0/0" {
					t.Errorf("AllowedIPs = %q", r.AllowedIPs)
				}
			},
		},
		{
			name: "with_dns",
			content: `[Interface]
PrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=
Address = 10.66.38.216/32
DNS = 10.64.0.1, 8.8.8.8

[Peer]
PublicKey = 7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=
Endpoint = 185.210.220.2:51820
AllowedIPs = 0.0.0.0/0, ::/0
`,
			checkResult: func(t *testing.T, r *wireguardConfigParsed) {
				if len(r.DNS) != 2 {
					t.Fatalf("DNS = %v, want 2 entries", r.DNS)
				}
				if r.DNS[0] != "10.64.0.1" || r.DNS[1] != "8.8.8.8" {
					t.Errorf("DNS = %v", r.DNS)
				}
			},
		},
		{
			name: "with_preshared_key",
			content: `[Interface]
PrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=
Address = 10.0.0.2/24

[Peer]
PublicKey = 7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=
PresharedKey = F3JcsRyn9Hywwyhl4EznlV4ZThatbB5Hi4U9b3emM+g=
Endpoint = 192.0.2.1:51820
AllowedIPs = 0.0.0.0/0
`,
			checkResult: func(t *testing.T, r *wireguardConfigParsed) {
				if r.PresharedKey != "F3JcsRyn9Hywwyhl4EznlV4ZThatbB5Hi4U9b3emM+g=" {
					t.Errorf("PresharedKey = %q", r.PresharedKey)
				}
			},
		},
		{
			name: "missing_public_key",
			content: `[Interface]
PrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=
Address = 10.0.0.2/24

[Peer]
Endpoint = 192.0.2.1:51820
`,
			wantErr: true,
		},
		{
			name: "missing_endpoint",
			content: `[Interface]
PrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=
Address = 10.0.0.2/24

[Peer]
PublicKey = 7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=
AllowedIPs = 0.0.0.0/0
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseWireGuardConfig(tc.content)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseWireGuardConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.checkResult != nil && result != nil {
				tc.checkResult(t, result)
			}
		})
	}
}

func TestParseWireGuardBase64Config(t *testing.T) {
	content := "[Interface]\nPrivateKey = WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=\nAddress = 10.0.0.2/24\n\n[Peer]\nPublicKey = 7B+2Z3odPbDNsfVr+F8invj6/mBKLVaolOHXZoCaBA0=\nEndpoint = 192.0.2.1:51820\nAllowedIPs = 0.0.0.0/0\n"
	b64 := base64.StdEncoding.EncodeToString([]byte(content))

	result, err := parseWireGuardBase64Config(b64)
	if err != nil {
		t.Fatalf("parseWireGuardBase64Config() error = %v", err)
	}

	if result.PrivateKey != "WPiBa/Ak1W+8Sp8L5yvbyhHeRO2o5kJvihq2VtJ+kFg=" {
		t.Errorf("PrivateKey = %q", result.PrivateKey)
	}
	if result.EndpointIP != "192.0.2.1" {
		t.Errorf("EndpointIP = %q", result.EndpointIP)
	}
	if result.EndpointPort != 51820 {
		t.Errorf("EndpointPort = %d", result.EndpointPort)
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		input    string
		wantIP   string
		wantPort int64
		wantErr  bool
	}{
		{"192.0.2.1:51820", "192.0.2.1", 51820, false},
		{"example.com:51820", "example.com", 51820, false},
		{"[::1]:51820", "::1", 51820, false},
		{"no-port", "", 0, true},
		{"host:notanumber", "", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			ip, port, err := parseEndpoint(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseEndpoint(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if ip != tc.wantIP {
				t.Errorf("ip = %q, want %q", ip, tc.wantIP)
			}
			if port != tc.wantPort {
				t.Errorf("port = %d, want %d", port, tc.wantPort)
			}
		})
	}
}
