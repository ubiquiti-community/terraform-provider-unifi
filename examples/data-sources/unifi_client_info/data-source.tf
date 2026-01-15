data "unifi_client_info" "example" {
  mac = "aa:bb:cc:dd:ee:ff"
}

# Access client information
output "client_name" {
  value = data.unifi_client_info.example.name
}

output "client_ip" {
  value = data.unifi_client_info.example.ip
}

# Retrieve client from specific site
data "unifi_client_info" "site_specific" {
  site = "production"
  mac  = "11:22:33:44:55:66"
}
