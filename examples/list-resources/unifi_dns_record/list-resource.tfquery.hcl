# List all DNS records in the default site
list "unifi_dns_record" "all" {
  provider = unifi
}

# List DNS records in a specific site
list "unifi_dns_record" "site_records" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List only A records
list "unifi_dns_record" "a_records" {
  provider = unifi

  config {
    filter {
      name  = "record_type"
      value = "A"
    }
  }
}

# List only enabled DNS records
list "unifi_dns_record" "enabled" {
  provider = unifi

  config {
    filter {
      name  = "enabled"
      value = "true"
    }
  }
}

# List a DNS record by name
list "unifi_dns_record" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "host.example.com"
    }
  }
}
