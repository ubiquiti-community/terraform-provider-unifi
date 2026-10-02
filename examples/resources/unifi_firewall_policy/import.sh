# Firewall policies can be imported using the policy ID, or site:id for a non-default site.
terraform import unifi_firewall_policy.lan_to_dmz_https 6512a1f04ee8cb0f1f4a9876

# For a non-default site, prefix the ID with the site name and a colon.
terraform import unifi_firewall_policy.lan_to_dmz_https default:6512a1f04ee8cb0f1f4a9876
