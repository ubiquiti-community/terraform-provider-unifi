# Example: Managing UniFi Settings with opt-in configuration

# Configure only management settings
resource "unifi_setting" "mgmt_only" {
  site = "default"

  mgmt = {
    auto_upgrade = {
      enabled = true
      hour    = 3
    }
    ssh = {
      enabled = true
      keys = [{
        name    = "admin-key"
        type    = "ssh-rsa"
        key     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD... admin@example.com"
        comment = "Administrator SSH Key"
      }]
    }
  }
}

# Configure multiple settings types
resource "unifi_setting" "combined" {
  site = "default"

  mgmt = {
    auto_upgrade = { enabled = true }
    ssh          = { enabled = false }
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
    ftp_module     = false

    upnp = {
      enabled       = true
      wan_interface = "WAN"
    }

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
