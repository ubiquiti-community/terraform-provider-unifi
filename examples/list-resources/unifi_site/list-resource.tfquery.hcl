# List all sites in the UniFi controller (sites are global, not site-scoped)
list "unifi_site" "all" {
  provider = unifi
}

# List a site by name
list "unifi_site" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "default"
    }
  }
}

# List a site by description
list "unifi_site" "by_description" {
  provider = unifi

  config {
    filter {
      name  = "description"
      value = "Headquarters"
    }
  }
}
