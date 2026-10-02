# List all devices in the default site
list "unifi_device" "all" {
  provider = unifi
}

# List devices in a specific site
list "unifi_device" "site_devices" {
  provider = unifi

  config {
    site = "my-site"
  }
}

# List a device by name
list "unifi_device" "by_name" {
  provider = unifi

  config {
    filter {
      name  = "name"
      value = "Office Switch"
    }
  }
}

# List a device by MAC address
list "unifi_device" "by_mac" {
  provider = unifi

  config {
    filter {
      name  = "mac"
      value = "00:11:22:33:44:55"
    }
  }
}

# List devices by model
list "unifi_device" "by_model" {
  provider = unifi

  config {
    filter {
      name  = "model"
      value = "US8P150"
    }
  }
}

# List devices by type (e.g. usw, uap, ugw)
list "unifi_device" "switches" {
  provider = unifi

  config {
    filter {
      name  = "type"
      value = "usw"
    }
  }
}
