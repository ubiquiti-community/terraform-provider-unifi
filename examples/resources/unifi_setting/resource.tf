# Example: Managing UniFi Settings with opt-in configuration

# Configure only management settings
resource "unifi_setting" "mgmt_only" {
  site = "default"

  mgmt = {
    auto_upgrade = true
    ssh_enabled  = true
    ssh_keys = [{
      name    = "admin-key"
      type    = "ssh-rsa"
      key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD... admin@example.com"
      comment = "Administrator SSH Key"
    }]
  }
}

# Configure multiple settings types
resource "unifi_setting" "combined" {
  site = "default"

  mgmt = {
    auto_upgrade = true
    ssh_enabled  = false
  }

  radius = {
    accounting_enabled      = true
    auth_port               = 1812
    acct_port               = 1813
    interim_update_interval = "10m"
    secret                  = "my-radius-secret"
  }

  usg = {
    broadcast_ping = false
    upnp_enabled   = true
    ftp_module     = false

    # DNS verification is a nested object on the USG/gateway settings.
    dns_verification = {
      domain             = "example.com"
      primary_dns_server = "1.1.1.1"
    }
  }
}

# Configure only RADIUS settings
resource "unifi_setting" "radius_only" {
  site = "default"

  radius = {
    accounting_enabled = true
    auth_port          = 1812
  }
}

# Configure Global Switch Settings (spanning tree, rogue DHCP detection,
# jumbo frames, 802.1X) for every switch on the site
resource "unifi_setting" "global_switch" {
  site = "default"

  global_switch = {
    stp_version            = "rstp"
    dhcp_snoop             = true
    jumboframe_enabled     = true
    dot1x_portctrl_enabled = false
  }
}

# Turn Wireless Meshing off on a fully wired site. Besides freeing the standby
# mesh radio, this doubles the per-band SSID budget: with meshing on, APs
# reserve a hidden backhaul SSID and allow 4 SSIDs per band instead of 8, so a
# site rebuilt from code can otherwise fail to create its fifth WLAN. The
# controller-generated mesh SSID and pre-shared key are left untouched.
resource "unifi_setting" "wired_site" {
  site = "default"

  connectivity = {
    enabled = false
  }
}
