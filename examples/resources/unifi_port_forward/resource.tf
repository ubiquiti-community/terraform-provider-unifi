# Basic port forward - HTTP to internal web server
resource "unifi_port_forward" "web_server" {
  name = "Web Server"

  wan = {
    port = "80"
  }

  forward = {
    ip   = "192.168.1.100"
    port = "8080"
  }
}

# HTTPS forward on a specific WAN interface
resource "unifi_port_forward" "https" {
  name     = "HTTPS Server"
  protocol = "tcp"

  wan = {
    interface = "wan"
    port      = "443"
  }

  forward = {
    ip   = "192.168.1.50"
    port = "443"
  }
}

# SSH access restricted to a specific source IP range
resource "unifi_port_forward" "ssh_restricted" {
  name = "SSH (Restricted)"

  wan = {
    port = "22"
  }

  forward = {
    ip   = "192.168.1.10"
    port = "22"
  }

  source_limiting = {
    ip      = "10.0.0.0/24"
    enabled = true
  }
}

# SSH access restricted using a firewall group
resource "unifi_firewall_group" "trusted_ips" {
  name = "trusted-ssh-sources"
  type = "address-group"
  members = [
    "10.0.0.1",
    "10.0.0.2",
  ]
}

resource "unifi_port_forward" "ssh_firewall_group" {
  name = "SSH (Firewall Group)"

  wan = {
    port = "2222"
  }

  forward = {
    ip   = "192.168.1.10"
    port = "22"
  }

  source_limiting = {
    firewall_group_id = unifi_firewall_group.trusted_ips.id
    enabled           = true
  }
}

# Port forward with syslog logging enabled
resource "unifi_port_forward" "game_server" {
  name           = "Game Server"
  protocol       = "udp"
  syslog_logging = true

  wan = {
    port = "27015"
  }

  forward = {
    ip   = "192.168.1.200"
    port = "27015"
  }
}
