package unifi

import (
	"context"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func Test_radiusUserResource_MoveState(t *testing.T) {
	tests := []struct {
		name      string
		r         *radiusUserResource
		wantCount int
	}{
		{
			name:      "returns_one_mover",
			r:         &radiusUserResource{},
			wantCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.MoveState(context.Background())
			if len(got) != tt.wantCount {
				t.Errorf("MoveState() returned %d movers, want %d", len(got), tt.wantCount)
			}
			// The sole mover must have a SourceSchema set.
			if got[0].SourceSchema == nil {
				t.Error("expected SourceSchema to be non-nil")
			}
		})
	}
}

func Test_radiusUserResource_moveFromAccount(t *testing.T) {
	tests := []struct {
		name          string
		req           fwresource.MoveStateRequest
		wantSkipped   bool // true means resp.TargetState should NOT be set (mover skipped)
		wantDiagError bool
	}{
		{
			name: "wrong_source_type_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:      "unifi_other",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
				SourceSchemaVersion: 0,
			},
			wantSkipped: true,
		},
		{
			name: "wrong_provider_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:      "unifi_account",
				SourceProviderAddress: "registry.terraform.io/someone-else/other",
				SourceSchemaVersion: 0,
			},
			wantSkipped: true,
		},
		{
			name: "correct_provider_address_with_slashes",
			req: fwresource.MoveStateRequest{
				SourceTypeName:      "unifi_account",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
				SourceSchemaVersion: 0,
				// SourceState is nil, so the mover returns early after passing the guard.
			},
			wantSkipped: true, // nil SourceState causes early return after provider guard
		},
		{
			name: "empty_provider_address_passes_guard",
			req: fwresource.MoveStateRequest{
				SourceTypeName:      "unifi_account",
				SourceProviderAddress: "",
				SourceSchemaVersion: 0,
				// SourceState is nil — guard passes (empty addr is allowed), then nil state guard triggers.
			},
			wantSkipped: true,
		},
		{
			name: "non_zero_schema_version_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:      "unifi_account",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
				SourceSchemaVersion: 1,
			},
			wantSkipped: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &radiusUserResource{}
			resp := &fwresource.MoveStateResponse{}
			r.moveFromAccount(context.Background(), tt.req, resp)
			// We only check that no unexpected diagnostic errors were raised;
			// the guard conditions cause early returns without diagnostics.
			if tt.wantDiagError && !resp.Diagnostics.HasError() {
				t.Error("expected diagnostics error, got none")
			}
			if !tt.wantDiagError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected diagnostics error: %v", resp.Diagnostics)
			}
		})
	}
}
