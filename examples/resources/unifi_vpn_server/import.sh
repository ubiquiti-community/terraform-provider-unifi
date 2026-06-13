# VPN servers can be imported by their 24-character hex network ID:
terraform import unifi_vpn_server.wireguard 5f3e1c2b9a8d7e6f5a4b3c2d

# Or by name, using the "name=" prefix:
terraform import unifi_vpn_server.wireguard name=wireguard

# Both forms may be prefixed with a site name and a colon to target a specific
# site (otherwise the provider's default site is used):
terraform import unifi_vpn_server.wireguard default:5f3e1c2b9a8d7e6f5a4b3c2d
terraform import unifi_vpn_server.wireguard default:name=wireguard
