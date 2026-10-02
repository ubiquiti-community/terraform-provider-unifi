# List all firewall policies in the default site
list "unifi_firewall_policy" "all" {
  provider = unifi
}

# List firewall policies in a specific site
list "unifi_firewall_policy" "site_policies" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List only enabled firewall policies
list "unifi_firewall_policy" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}

# List firewall policies by action
list "unifi_firewall_policy" "blocks" {
  provider = unifi

  config {
    filter {
      name  = "action"
      value = "BLOCK"
    }
  }
}

# List a firewall policy by name
list "unifi_firewall_policy" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Allow Web"
    }
  }
}
