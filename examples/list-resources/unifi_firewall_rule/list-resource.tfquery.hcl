# List all firewall rules in the default site
list "unifi_firewall_rule" "all" {
  provider = unifi
}

# List firewall rules in a specific site
list "unifi_firewall_rule" "site_rules" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List only enabled firewall rules
list "unifi_firewall_rule" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}

# List firewall rules by name
list "unifi_firewall_rule" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Allow Web"
    }
  }
}

# List firewall rules in a specific ruleset
list "unifi_firewall_rule" "wan_in" {
  provider = unifi

  config {
    filter {
      name  = "ruleset"
      value = "WAN_IN"
    }
  }
}

# List firewall rules by action
list "unifi_firewall_rule" "drops" {
  provider = unifi

  config {
    filter {
      name  = "action"
      value = "drop"
    }
  }
}
