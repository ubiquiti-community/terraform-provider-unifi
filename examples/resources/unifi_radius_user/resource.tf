# unifi_radius_user is the modern replacement for the deprecated unifi_account resource.

# Basic RADIUS user authenticating with a name and password.
resource "unifi_radius_user" "basic" {
  name     = "alice"
  password = var.radius_user_password
}

# MAC-based authentication (MAB) account with an explicit VLAN assignment.
# For MAC auth, use the uppercased MAC (no colons/periods) as both name and password.
resource "unifi_radius_user" "mac_auth" {
  name     = "A1B2C3D4E5F6"
  password = "A1B2C3D4E5F6"

  # RFC 2868 tunnel attributes for dynamic VLAN assignment.
  tunnel_type        = 13 # VLAN
  tunnel_medium_type = 6  # IEEE-802
  vlan               = 100
}

# User whose VLAN is inherited from an existing network. When vlan is omitted
# but network_id is set, the account derives its VLAN from that network.
resource "unifi_radius_user" "from_network" {
  name     = "bob"
  password = var.radius_user_password

  network_id         = unifi_network.vlan_users.id
  tunnel_config_type = "802.1x"
}
