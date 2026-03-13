---
page_title: Network Members Group List (Data Source)
subcategory: ""
description: |-
  Retrieves a list of all network members groups.
---

# Network Members Group List (Data Source)

Retrieves a list of all network members groups.

## Example Usage

```terraform
data "unifi_network_members_group_list" "all" {
  # Retrieves all network members groups from the default site
}

# Output the total number of groups
output "total_groups" {
  value = length(data.unifi_network_members_group_list.all.groups)
}
```
