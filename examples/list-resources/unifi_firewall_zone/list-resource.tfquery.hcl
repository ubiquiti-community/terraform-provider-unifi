# List all firewall zones in the default site
list "unifi_firewall_zone" "all" {
  provider = unifi
}

# List firewall zones in a specific site
list "unifi_firewall_zone" "site_zones" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List firewall zones by name
list "unifi_firewall_zone" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Internal"
    }
  }
}
