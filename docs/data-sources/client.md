---
page_title: Client (Data Source)
subcategory: ""
description: |-
  Retrieves properties of a client of the network by MAC address.
---

# Client (Data Source)

Retrieves properties of a client of the network by MAC address.

## Example Usage

```terraform
data "unifi_client" "default" {
  mac = "01:23:45:67:89:ab"
}
```
