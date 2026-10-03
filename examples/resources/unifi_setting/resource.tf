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

# Configure only SNMP settings (v1/v2c community plus an SNMPv3 user).
# The write-only password below requires Terraform 1.11+.
variable "snmp_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

resource "unifi_setting" "snmp_only" {
  site = "default"

  snmp = {
    enabled             = true
    community           = "my-snmp-community"
    enabled_v3          = true
    username            = "monitor"
    password_wo         = var.snmp_password
    password_wo_version = 1 # Change this version whenever the password changes.

    # Alternatively, replace the two password_wo fields with:
    # password = "my-snmpv3-password"
    # That value is stored in state and preserved on read. Neither password
    # path detects controller-side password changes. Community is read back.
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

# Client Device Isolation (Settings > Networks > "Device Isolation (ACL)") is a
# switch ACL listing the networks whose devices may not talk to each other. It
# covers same-network traffic across access points, which unifi_wlan's
# l2_isolation (one access point) and unifi_network's network_isolation (between
# networks) do not. The controller only offers it for networks routed by a UniFi
# gateway or L3 switch.
resource "unifi_setting" "device_isolation" {
  site = "default"

  global_switch = {
    acl_device_isolation = [
      unifi_network.guest.id,
      unifi_network.quarantine.id,
    ]
  }
}
