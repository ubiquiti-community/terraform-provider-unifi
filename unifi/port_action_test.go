package unifi

import (
	"context"
	"testing"

	fwaction "github.com/hashicorp/terraform-plugin-framework/action"
)

func TestNewPortAction(t *testing.T) {
	got := NewPortAction()
	if got == nil {
		t.Fatal("NewPortAction() returned nil")
	}
	if _, ok := got.(fwaction.ActionWithConfigure); !ok {
		t.Error("expected ActionWithConfigure interface")
	}
}

func Test_portAction_Metadata(t *testing.T) {
	tests := []struct {
		name         string
		providerType string
		wantTypeName string
	}{
		{
			name:         "sets_type_name",
			providerType: "unifi",
			wantTypeName: "unifi_port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &portAction{}
			resp := &fwaction.MetadataResponse{}
			a.Metadata(
				context.Background(),
				fwaction.MetadataRequest{ProviderTypeName: tt.providerType},
				resp,
			)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_portAction_Schema(t *testing.T) {
	tests := []struct {
		name       string
		wantAttrs  []string
	}{
		{
			name:      "has_required_attributes",
			wantAttrs: []string{"device_mac", "port_number", "poe_mode", "timeouts"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &portAction{}
			resp := &fwaction.SchemaResponse{}
			a.Schema(context.Background(), fwaction.SchemaRequest{}, resp)
			for _, attr := range tt.wantAttrs {
				if _, ok := resp.Schema.Attributes[attr]; !ok {
					t.Errorf("expected attribute %q in schema", attr)
				}
			}
		})
	}
}

func Test_portAction_Configure(t *testing.T) {
	tests := []struct {
		name      string
		req       fwaction.ConfigureRequest
		wantError bool
	}{
		{
			name:      "nil_provider_data",
			req:       fwaction.ConfigureRequest{},
			wantError: false,
		},
		{
			name:      "wrong_type",
			req:       fwaction.ConfigureRequest{ProviderData: "not-a-client"},
			wantError: true,
		},
		{
			name:      "correct_client",
			req:       fwaction.ConfigureRequest{ProviderData: &Client{}},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &portAction{}
			resp := &fwaction.ConfigureResponse{}
			a.Configure(context.Background(), tt.req, resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

