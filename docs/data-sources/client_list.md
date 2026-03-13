---
page_title: Client List (Data Source)
subcategory: ""
description: |-
  Retrieves a list of clients (users) on the network with optional filtering. Merges client configuration data with active and historical connection information for network discovery within Terraform.
---

# Client List (Data Source)

Retrieves a list of clients (users) on the network with optional filtering. Merges client configuration data with active and historical connection information for network discovery within Terraform.

## Example Usage

```terraform
# List all clients on the network
data "unifi_client_list" "all" {
}

# List only wired clients
data "unifi_client_list" "wired" {
  wired = true
}

# List clients in a specific network members group
data "unifi_client_list" "iot_devices" {
  group = "IoT Devices"
}

# List non-blocked clients from a specific vendor
data "unifi_client_list" "apple_devices" {
  oui     = "Apple"
  blocked = false
}

# Output client count
output "total_clients" {
  value = length(data.unifi_client_list.all.clients)
}

# Output wired client IPs
output "wired_client_ips" {
  value = [for c in data.unifi_client_list.wired.clients : c.ip if c.ip != null]
}
```
