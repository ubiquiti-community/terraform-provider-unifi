# Read the Teleport settings of a site (defaults to the provider's site).
data "unifi_teleport" "this" {}

output "teleport_enabled" {
  description = "Whether Teleport is enabled."
  value       = data.unifi_teleport.this.enabled
}

output "teleport_subnet_cidr" {
  description = "The subnet used for Teleport, or null when none is set."
  value       = data.unifi_teleport.this.subnet_cidr
}
