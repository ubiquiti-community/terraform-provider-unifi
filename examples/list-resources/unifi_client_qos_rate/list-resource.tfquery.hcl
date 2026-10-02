# List all client QOS rates in the default site
list "unifi_client_qos_rate" "all" {
  provider = unifi
}

# List client QOS rates in a specific site
list "unifi_client_qos_rate" "site_rates" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List a client QOS rate by name
list "unifi_client_qos_rate" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "guest-limit"
    }
  }
}
