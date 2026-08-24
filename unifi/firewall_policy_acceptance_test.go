package unifi

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccFirewallPolicy_scheduleRoundTrip(t *testing.T) {
	if os.Getenv("UNIFI_SKIP_CONTAINER") == "" {
		t.Skip(
			"firewall policy schedules require a real zone-based firewall controller; " +
				"set UNIFI_SKIP_CONTAINER to run",
		)
	}

	name := acctest.RandomWithPrefix("tf-acc-firewall-schedule")
	const resourceName = "unifi_firewall_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			preCheck(t)
			// Policies live on firewall zones; skip on controllers that
			// cannot manage zones (e.g. the dockerized demo controller).
			testAccFirewallZonePreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccFirewallPolicyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallPolicyScheduleConfig(name, "before schedule update"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					testAccSetFirewallPolicySchedule(resourceName),
				),
			},
			{
				Config: testAccFirewallPolicyScheduleConfig(name, "after schedule update"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName,
						"description",
						"after schedule update",
					),
					resource.TestCheckResourceAttr(resourceName, "schedule.mode", "EVERY_WEEK"),
					resource.TestCheckResourceAttr(
						resourceName,
						"schedule.time_all_day",
						"false",
					),
					resource.TestCheckResourceAttr(
						resourceName,
						"schedule.time_range_start",
						"09:00",
					),
					resource.TestCheckResourceAttr(
						resourceName,
						"schedule.time_range_end",
						"17:30",
					),
					resource.TestCheckResourceAttr(
						resourceName,
						"schedule.repeat_on_days.#",
						"3",
					),
					resource.TestCheckTypeSetElemAttr(
						resourceName,
						"schedule.repeat_on_days.*",
						"mon",
					),
					resource.TestCheckTypeSetElemAttr(
						resourceName,
						"schedule.repeat_on_days.*",
						"wed",
					),
					resource.TestCheckTypeSetElemAttr(
						resourceName,
						"schedule.repeat_on_days.*",
						"fri",
					),
				),
			},
			{
				Config:   testAccFirewallPolicyScheduleConfig(name, "after schedule update"),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccFirewallPolicy_import covers both import entry paths: the classic
// string import (by controller object id) and the Terraform 1.12+ import
// block with a resource identity.
func TestAccFirewallPolicy_import(t *testing.T) {
	if os.Getenv("UNIFI_SKIP_CONTAINER") == "" {
		t.Skip(
			"firewall policies require a real zone-based firewall controller; " +
				"set UNIFI_SKIP_CONTAINER to run",
		)
	}

	name := acctest.RandomWithPrefix("tf-acc-firewall-import")
	const resourceName = "unifi_firewall_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			preCheck(t)
			// Policies live on firewall zones; skip on controllers that
			// cannot manage zones (e.g. the dockerized demo controller).
			testAccFirewallZonePreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccFirewallPolicyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallPolicyScheduleConfig(name, "import test"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
			{
				Config:   testAccFirewallPolicyScheduleConfig(name, "import test"),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccSetFirewallPolicySchedule(resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found in state", resourceName)
		}

		ctx := context.Background()
		client, err := testAccFirewallPolicyClient(ctx)
		if err != nil {
			return err
		}

		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = "default"
		}
		policy, err := client.GetFirewallPolicy(ctx, site, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("read firewall policy before schedule update: %w", err)
		}

		timeAllDay := false
		policy.Schedule = &api.FirewallPolicySchedule{
			Mode:           "EVERY_WEEK",
			RepeatOnDays:   []string{"mon", "wed", "fri"},
			TimeAllDay:     &timeAllDay,
			TimeRangeStart: "09:00",
			TimeRangeEnd:   "17:30",
		}
		if _, err := client.UpdateFirewallPolicy(ctx, site, policy); err != nil {
			return fmt.Errorf("set firewall policy schedule through the API: %w", err)
		}
		return nil
	}
}

func testAccFirewallPolicyCheckDestroy(state *terraform.State) error {
	ctx := context.Background()
	client, err := testAccFirewallPolicyClient(ctx)
	if err != nil {
		return nil //nolint:nilerr // The test framework already reports destroy failures.
	}

	for _, rs := range state.RootModule().Resources {
		if rs.Type != "unifi_firewall_policy" {
			continue
		}
		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = "default"
		}
		if _, err := client.GetFirewallPolicy(ctx, site, rs.Primary.ID); err == nil {
			return fmt.Errorf("unifi_firewall_policy %s still exists", rs.Primary.ID)
		} else if _, ok := err.(*api.NotFoundError); !ok {
			return err
		}
	}
	return nil
}

func testAccFirewallPolicyClient(ctx context.Context) (*api.ApiClient, error) {
	return api.New(ctx, &api.Config{
		BaseURL:       os.Getenv("UNIFI_API"),
		Username:      os.Getenv("UNIFI_USERNAME"),
		Password:      os.Getenv("UNIFI_PASSWORD"),
		AllowInsecure: true,
	})
}

func testAccFirewallPolicyScheduleConfig(name, description string) string {
	return fmt.Sprintf(`
resource "unifi_firewall_zone" "source" {
  name        = %[1]q
  network_ids = []
}

resource "unifi_firewall_zone" "destination" {
  name        = "%[1]s-destination"
  network_ids = []
}

resource "unifi_firewall_policy" "test" {
  name        = %[1]q
  action      = "BLOCK"
  protocol    = "tcp"
  description = %[2]q
  enabled     = false

  source = {
    zone_id         = unifi_firewall_zone.source.id
    matching_target = "ANY"
  }

  destination = {
    zone_id            = unifi_firewall_zone.destination.id
    matching_target    = "ANY"
    port               = "65535"
    port_matching_type = "SPECIFIC"
  }
}
`, name, description)
}
