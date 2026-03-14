# List all traffic routes in the default site
list "unifi_traffic_route" "all" {
  provider = unifi
}

# List traffic routes in a specific site
list "unifi_traffic_route" "site_routes" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List only enabled traffic routes
list "unifi_traffic_route" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}

# List traffic routes targeting a specific network
list "unifi_traffic_route" "by_network" {
  provider = unifi

  config {
    filter {
      name  = "network_id"
      value = "6606e3e415f6df0721014c52"
    }
  }
}

# List traffic routes by matching target type
list "unifi_traffic_route" "domains_only" {
  provider = unifi

  config {
    filter {
      name  = "matching_target"
      value = "DOMAIN"
    }
  }
}
