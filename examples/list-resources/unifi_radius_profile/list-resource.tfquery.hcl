# List all RADIUS profiles in the default site
list "unifi_radius_profile" "all" {
  provider = unifi
}

# List RADIUS profiles in a specific site
list "unifi_radius_profile" "site_profiles" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List a RADIUS profile by name
list "unifi_radius_profile" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Default"
    }
  }
}
