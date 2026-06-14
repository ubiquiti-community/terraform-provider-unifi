package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewDeprecatedAccountResource(t *testing.T) {
	r := NewDeprecatedAccountResource()
	if r == nil {
		t.Fatal("NewDeprecatedAccountResource() returned nil")
	}
}

func Test_deprecatedAccountResource_Metadata(t *testing.T) {
	r := &deprecatedAccountResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "unifi"}, resp)
	if resp.TypeName != "unifi_account" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_account")
	}
}

func Test_deprecatedAccountResource_Schema(t *testing.T) {
	r := &deprecatedAccountResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	if resp.Schema.DeprecationMessage == "" {
		t.Error("expected a deprecation message")
	}
}

func TestNewDeprecatedAccountDataSource(t *testing.T) {
	d := NewDeprecatedAccountDataSource()
	if d == nil {
		t.Fatal("NewDeprecatedAccountDataSource() returned nil")
	}
}

func Test_deprecatedAccountDataSource_Metadata(t *testing.T) {
	d := &deprecatedAccountDataSource{}
	resp := &fwdatasource.MetadataResponse{}
	d.Metadata(context.Background(), fwdatasource.MetadataRequest{ProviderTypeName: "unifi"}, resp)
	if resp.TypeName != "unifi_account" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_account")
	}
}

func Test_deprecatedAccountDataSource_Schema(t *testing.T) {
	d := &deprecatedAccountDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	if resp.Schema.DeprecationMessage == "" {
		t.Error("expected a deprecation message")
	}
}
