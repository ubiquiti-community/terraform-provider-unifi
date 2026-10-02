# List all power supervisors in the default site
list "unifi_power_supervisor" "all" {
  provider = unifi
}

# List power supervisors in a specific site
list "unifi_power_supervisor" "site_supervisors" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List the power supervisor for a specific supervised device MAC
list "unifi_power_supervisor" "by_device" {
  provider = unifi

  config {
    filter {
      name  = "device_mac"
      value = "9c:05:d6:11:22:33"
    }
  }
}
