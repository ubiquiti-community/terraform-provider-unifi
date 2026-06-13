# RADIUS profile with external authentication and accounting servers,
# interim accounting updates, and VLAN assignment enabled.
resource "unifi_radius_profile" "external" {
  name = "external-radius"

  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = 3600

  vlan_enabled   = true
  vlan_wlan_mode = "required"

  auth_server {
    ip     = "10.0.0.10"
    port   = 1812
    secret = var.radius_auth_secret
  }

  acct_server {
    ip     = "10.0.0.10"
    port   = 1813
    secret = var.radius_acct_secret
  }
}

# Profile that uses the built-in USG/UDM as the RADIUS auth and accounting server.
resource "unifi_radius_profile" "usg" {
  name = "usg-radius"

  use_usg_auth_server = true
  use_usg_acct_server = true
  accounting_enabled  = true

  vlan_enabled   = true
  vlan_wlan_mode = "optional"
}

# A WPA-Enterprise WLAN can reference the profile via radius_profile_id.
resource "unifi_wlan" "corp" {
  name       = "corp-wpaeap"
  security   = "wpaeap"
  network_id = unifi_network.corp.id

  radius_profile_id = unifi_radius_profile.external.id
}
