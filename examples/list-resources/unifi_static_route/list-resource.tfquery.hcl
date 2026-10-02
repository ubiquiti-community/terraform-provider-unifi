# List all static routes in the default site
list "unifi_static_route" "all" {
  provider = unifi
}

# List static routes in a specific site
list "unifi_static_route" "site_routes" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List static routes by name
list "unifi_static_route" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "To Branch"
    }
  }
}

# List static routes by type
list "unifi_static_route" "nexthop_only" {
  provider = unifi

  config {
    filter {
      name  = "type"
      value = "nexthop-route"
    }
  }
}
