---
page_title: Client Info List (Data Source)
subcategory: ""
description: |-
  Retrieves a list of all active clients on the network.
---

# Client Info List (Data Source)

Retrieves a list of all active clients on the network.

## Example Usage

```terraform
data "unifi_client_info_list" "all" {
  # Retrieves all active clients from the default site
}

# Get total client count
output "total_clients" {
  value = length(data.unifi_client_info_list.all.clients)
}
```
