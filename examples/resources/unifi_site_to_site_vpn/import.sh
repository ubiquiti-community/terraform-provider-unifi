# Site-to-site VPN networks can be imported using the network ID (the pre-shared
# key is not recovered — set it in config and re-apply).
terraform import unifi_site_to_site_vpn.hq_to_branch 5f7e8d9c0a1b2c3d4e5f6a7b

# Or with an explicit site:
terraform import unifi_site_to_site_vpn.hq_to_branch default:5f7e8d9c0a1b2c3d4e5f6a7b
