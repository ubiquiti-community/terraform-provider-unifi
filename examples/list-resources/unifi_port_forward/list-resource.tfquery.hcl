# List all port forwarding rules in the default site
list "unifi_port_forward" "all" {
  provider = unifi
}

# List port forwarding rules in a specific site
list "unifi_port_forward" "site_rules" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List only enabled port forwarding rules
list "unifi_port_forward" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}

# List port forwarding rules by name
list "unifi_port_forward" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "HTTPS"
    }
  }
}
