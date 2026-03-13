---
page_title: Network (Data Source)
subcategory: ""
description: |-
  unifi_network data source can be used to retrieve settings for a network by name or ID.
---

# Network (Data Source)

`unifi_network` data source can be used to retrieve settings for a network by name or ID.

## Example Usage

```terraform
#retrieve network data by unifi network name
data "unifi_network" "lan_network" {
  name = "Default"
}

#retrieve network data from user record
data "unifi_client" "my_device" {
  mac = "01:23:45:67:89:ab"
}
data "unifi_network" "my_network" {
  id = data.unifi_client.my_device.network_id
}
```
