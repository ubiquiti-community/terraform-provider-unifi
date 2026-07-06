# AP groups can be imported using the group ID, or site:id for a non-default site.
terraform import unifi_ap_group.example 5f3e9b2c4ee8cb0f1f4a1234

# For a non-default site, prefix the ID with the site name and a colon.
terraform import unifi_ap_group.example default:5f3e9b2c4ee8cb0f1f4a1234
