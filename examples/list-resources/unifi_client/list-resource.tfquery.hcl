# List all clients in the default site
list "unifi_client" "all" {
  provider = unifi
}

# List clients in a specific site
list "unifi_client" "site_clients" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List clients filtered by group name
list "unifi_client" "by_group" {
  provider = unifi

  config {
    group = "my-group"
  }
}

# List clients filtered by network name
list "unifi_client" "by_network" {
  provider = unifi

  config {
    filter {
      name  = "network_name"
      value = "wifi-vlan"
    }
  }
}

# List only wired clients
list "unifi_client" "wired_only" {
  provider = unifi

  config {
    filter {
      name  = "is_wired"
      value = "true"
    }
  }
}
