# List all dynamic DNS configurations in the default site
list "unifi_dynamic_dns" "all" {
  provider = unifi
}

# List dynamic DNS configurations in a specific site
list "unifi_dynamic_dns" "site_entries" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List dynamic DNS configurations by host name
list "unifi_dynamic_dns" "by_host" {
  provider = unifi

  config {
    filter {
      name  = "host_name"
      value = "home.example.com"
    }
  }
}

# List dynamic DNS configurations by service provider
list "unifi_dynamic_dns" "by_service" {
  provider = unifi

  config {
    filter {
      name  = "service"
      value = "dyndns"
    }
  }
}
