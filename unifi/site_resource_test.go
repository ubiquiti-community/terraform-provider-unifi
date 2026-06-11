package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_site.test", "description", "tfacc-test"),
					resource.TestCheckResourceAttrSet("unifi_site.test", "name"),
				),
				ResourceName:  "unifi_site.test",
				ImportState:   true,
				ImportStateId: "default",
			},
		},
	})
}

func testAccSiteFrameworkConfig_basic() string {
	return `
resource "unifi_site" "test" {
	name        = "default"
	description = "tfacc-test"
}
`
}

// TestSiteToModelNilDoesNotPanic guards #261: siteToModel must return an error
// for a nil site instead of dereferencing it (the read path used to fall
// through to a nil siteToModel on a not-found, panicking the provider).
func TestSiteToModelNilDoesNotPanic(t *testing.T) {
	r := &siteFrameworkResource{}
	var model siteFrameworkResourceModel
	diags := r.siteToModel(context.Background(), nil, &model)
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for a nil site, got none")
	}
}
