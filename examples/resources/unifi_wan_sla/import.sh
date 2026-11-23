# WAN SLA profiles can be imported using the profile ID (Network 11 uses UUIDs), or site:id for a non-default site.
terraform import unifi_wan_sla.standard 00000000-0000-4000-8000-000000000002

# For a non-default site, prefix the ID with the site name and a colon.
terraform import unifi_wan_sla.standard default:00000000-0000-4000-8000-000000000002
