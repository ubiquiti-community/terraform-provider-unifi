# List all port profiles in the default site
list "unifi_port_profile" "all" {
  provider = unifi
}

# List port profiles in a specific site
list "unifi_port_profile" "site_profiles" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List port profiles by name
list "unifi_port_profile" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Uplink"
    }
  }
}
