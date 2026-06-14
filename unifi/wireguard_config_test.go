package unifi

import (
	"encoding/base64"
	"reflect"
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

func Test_parseWireGuardConfig(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name    string
		args    args
		want    *wireguardConfigParsed
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWireGuardConfig(tt.args.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWireGuardConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseWireGuardConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseEndpoint(t *testing.T) {
	type args struct {
		endpoint string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseEndpoint(tt.args.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseEndpoint() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseEndpoint() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_parseWireGuardBase64Config(t *testing.T) {
	type args struct {
		b64Content string
	}
	tests := []struct {
		name    string
		args    args
		want    *wireguardConfigParsed
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWireGuardBase64Config(tt.args.b64Content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWireGuardBase64Config() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseWireGuardBase64Config() = %v, want %v", got, tt.want)
			}
		})
	}
}
