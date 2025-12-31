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
    interim_update_interval = 600
    secret                  = "my-radius-secret"
  }

  usg = {
    multicast_dns_enabled = true
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
