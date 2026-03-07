# List all wired clients in the "Servers" group and bulk-import them as
# unifi_client resources.  Run:
#
#   terraform query -generate-config-out=generated.tf
#
# to produce the import blocks and resource skeletons, then move the generated
# resources into your main configuration.

list "unifi_client" "servers" {
  provider = unifi

  config {
    # Defaults to the provider's configured site when omitted.
    site  = "default"

    # Resolved to a group ID before querying the API.
    group = "Servers"

    # Generic filter blocks — pass API field/value pairs directly.
    # Multiple filter blocks are AND-ed; values within a block are OR-ed.
    filter {
      name   = "is_wired"
      values = ["true"]
    }
  }
}
