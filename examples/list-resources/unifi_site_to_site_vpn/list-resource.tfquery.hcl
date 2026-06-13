# List all site-to-site VPNs in the default site
list "unifi_site_to_site_vpn" "all" {
  provider = unifi
}

# List site-to-site VPNs in a specific site
list "unifi_site_to_site_vpn" "site_vpns" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List site-to-site VPNs filtered by name
list "unifi_site_to_site_vpn" "filtered" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "branch-office"
    }
  }
}
