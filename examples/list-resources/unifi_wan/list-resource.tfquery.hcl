# List all WAN networks in the default site
list "unifi_wan" "all" {
  provider = unifi
}

# List WAN networks in a specific site
list "unifi_wan" "site_wans" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List WAN networks filtered by name
list "unifi_wan" "filtered" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "WAN"
    }
  }
}
