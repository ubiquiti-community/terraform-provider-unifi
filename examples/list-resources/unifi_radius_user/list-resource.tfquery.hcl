# List all RADIUS users in the default site
list "unifi_radius_user" "all" {
  provider = unifi
}

# List RADIUS users in a specific site
list "unifi_radius_user" "site_users" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List a RADIUS user by name
list "unifi_radius_user" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "AABBCCDDEEFF"
    }
  }
}
