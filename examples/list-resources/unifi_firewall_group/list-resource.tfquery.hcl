# List all firewall groups in the default site
list "unifi_firewall_group" "all" {
  provider = unifi
}

# List firewall groups in a specific site
list "unifi_firewall_group" "site_groups" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List firewall groups by name
list "unifi_firewall_group" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Web Servers"
    }
  }
}

# List firewall groups by type
list "unifi_firewall_group" "address_groups" {
  provider = unifi

  config {
    filter {
      name  = "type"
      value = "address-group"
    }
  }
}
