# List all WLANs in the default site
list "unifi_wlan" "all" {
  provider = unifi
}

# List WLANs in a specific site
list "unifi_wlan" "site_wlans" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List WLANs filtered by name
list "unifi_wlan" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "guest-wifi"
    }
  }
}

# List only enabled WLANs
list "unifi_wlan" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}
