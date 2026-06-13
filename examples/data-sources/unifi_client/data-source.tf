# Look up a known client by its MAC address. Most attributes (name, fixed IP,
# blocked state, QoS rate group, etc.) are computed from the controller.
data "unifi_client" "workstation" {
  mac = "01:23:45:67:89:ab"
}

output "client_fixed_ip" {
  description = "The fixed IPv4 address assigned to the client, if any."
  value       = data.unifi_client.workstation.fixed_ip
}

output "client_blocked" {
  description = "Whether the client is currently blocked from the network."
  value       = data.unifi_client.workstation.blocked
}

output "client_qos_max_down" {
  description = "Maximum download rate (kbps) from the client's QoS group."
  value       = data.unifi_client.workstation.qos_rate.max_down
}
