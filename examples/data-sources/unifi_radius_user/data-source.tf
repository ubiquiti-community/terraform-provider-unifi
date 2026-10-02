# Look up a RADIUS user account by name.
data "unifi_radius_user" "example" {
  name = "vpn-user"
}

# The network this account is bound to (computed).
output "radius_user_network_id" {
  value = data.unifi_radius_user.example.network_id
}

# RFC2868 tunnel attributes returned by the controller (computed).
output "radius_user_tunnel_type" {
  value = data.unifi_radius_user.example.tunnel_type
}

# Look up an account on a specific site.
data "unifi_radius_user" "site_specific" {
  site = "production"
  name = "wifi-user"
}
