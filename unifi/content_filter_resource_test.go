package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func Test_contentFilterResource_Schema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&contentFilterResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced errors: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("schema invalid: %v", diags)
	}
}

func Test_contentFilterResource_roundTrip(t *testing.T) {
	r, ctx := &contentFilterResource{}, context.Background()
	// As UniFi Network 10.6 returns it: unused lists come back empty.
	live := unifi.ContentFiltering{
		ID:         "6ab63bde417220a6e4272d49",
		Name:       "adblocking",
		Enabled:    true,
		NetworkIDs: []string{"66ec38ab34dbcb19dce4a67f"},
		Categories: []string{"ADVERTISEMENT"},
		ClientMACs: []string{},
		AllowList:  []string{},
		BlockList:  []string{},
		SafeSearch: []string{},
		Schedule:   &unifi.ContentFilteringSchedule{Mode: "ALWAYS"},
	}
	var diags diag.Diagnostics
	var m contentFilterModel
	r.apiToModel(ctx, &live, &m, "default", &diags)
	if !m.AllowList.IsNull() || !m.SafeSearch.IsNull() {
		t.Error("empty lists should read as null")
	}
	got := r.modelToAPI(ctx, &m, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	want := unifi.ContentFiltering{
		Name: "adblocking", Enabled: true,
		NetworkIDs: live.NetworkIDs, Categories: live.Categories,
		Schedule: &unifi.ContentFilteringSchedule{Mode: "ALWAYS"},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("round trip =\n%+v\nwant\n%+v", *got, want)
	}
}
