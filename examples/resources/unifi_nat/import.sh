# NAT rules can be imported using the rule ID.
terraform import unifi_nat.dnat 6512a1f04ee8cb0f1f4a9876

# For a non-default site, prefix the ID with the site name and a colon.
terraform import unifi_nat.dnat default:6512a1f04ee8cb0f1f4a9876
