package unifi

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// The controller stores no tagged_networkconf_ids on a port override (checked
// on UniFi Network 9.5.21, 10.0.162 and 11.0.103): it
// drops the list from a PUT and keeps excluded_networkconf_ids instead. With
// tagged_vlan_mgmt = "custom" the port tags every taggable network that is not
// excluded, and the port's native network cannot be excluded
// (api.err.NativeVlanCannotBeExcluded). So a configured tagged set is written
// as its complement, and read back from the complement when the controller
// returns no tagged list. Without that, a "custom" port tagged every network
// of the site while the plan said it tagged only the configured ones.

// taggableNetworkIDs returns the networks the controller can tag on a switch
// port: the LAN networks. VPN and WAN networks are never in a port's lists.
func taggableNetworkIDs(ctx context.Context, client *Client, site string) ([]string, error) {
	networks, err := client.ListNetwork(ctx, site)
	if err != nil {
		return nil, fmt.Errorf("listing the networks to tag on the port: %w", err)
	}
	ids := make([]string, 0, len(networks))
	for _, n := range networks {
		switch n.Purpose {
		case "corporate", "guest", "vlan-only":
			ids = append(ids, n.ID)
		}
	}
	slices.Sort(ids)
	return ids, nil
}

// networksExcept returns the sorted networks of taggable that are neither in
// omit nor native.
func networksExcept(taggable, omit []string, native string) []string {
	out := []string{}
	for _, id := range taggable {
		if id != native && !slices.Contains(omit, id) {
			out = append(out, id)
		}
	}
	return out
}

// excludeUntaggedNetworks sets excluded_networkconf_ids on every declared
// override that configures tagged_networkconf_ids: every taggable network
// except the tagged ones and the port's native network, which is the
// declared one or else the one the controller holds in current. A declared
// set equal to the controller's keeps the controller's order, so an unchanged
// port marshals identically and is not written.
func excludeUntaggedNetworks(
	current, declared []unifi.DevicePortOverrides,
	taggable []string,
) {
	currentByIdx := make(map[int64]unifi.DevicePortOverrides, len(current))
	for _, po := range current {
		if po.PortIDX != nil {
			currentByIdx[*po.PortIDX] = po
		}
	}
	for i := range declared {
		po := &declared[i]
		if po.TaggedNetworkIDs == nil {
			continue
		}
		var cur unifi.DevicePortOverrides
		if po.PortIDX != nil {
			cur = currentByIdx[*po.PortIDX]
		}
		native := po.NATiveNetworkID
		if native == "" {
			native = cur.NATiveNetworkID
		}
		excluded := networksExcept(taggable, po.TaggedNetworkIDs, native)
		if sameStringSet(excluded, cur.ExcludedNetworkIDs) {
			excluded = cur.ExcludedNetworkIDs
		}
		po.ExcludedNetworkIDs = excluded
	}
}

// taggedNetworksFromAPI returns the networks the port tags. A controller that
// stores tagged_networkconf_ids returns them; otherwise (every version checked) they
// follow from tagged_vlan_mgmt: none for block_all, every taggable network
// except the excluded ones for custom, and every one for auto. The native
// network is untagged and never counted.
func taggedNetworksFromAPI(po unifi.DevicePortOverrides, taggable []string) []string {
	if len(po.TaggedNetworkIDs) > 0 || taggable == nil {
		return po.TaggedNetworkIDs
	}
	switch po.TaggedVLANMgmt {
	case "block_all":
		return []string{}
	case "custom":
		return networksExcept(taggable, po.ExcludedNetworkIDs, po.NATiveNetworkID)
	default:
		return networksExcept(taggable, nil, po.NATiveNetworkID)
	}
}

// portOverridesSetTagged reports whether any port_override block configures
// tagged_networkconf_ids, the only case that needs the networks of the site.
func portOverridesSetTagged(ctx context.Context, set types.Set) bool {
	var models []portOverrideModel
	if set.ElementsAs(ctx, &models, false).HasError() {
		return false
	}
	return slices.ContainsFunc(models, func(m portOverrideModel) bool {
		return isKnown(m.TaggedNetworkIDs)
	})
}
