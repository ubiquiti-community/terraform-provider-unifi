# Look up a client QoS rate limit group (client group) by name. The configured
# upload/download caps are computed, useful for reporting or wiring into other
# configuration.
data "unifi_client_qos_rate" "wifi" {
  name = "wifi"
}

output "qos_rate_max_down" {
  description = "The maximum download rate (kbps) for the QoS group."
  value       = data.unifi_client_qos_rate.wifi.qos_rate_max_down
}

output "qos_rate_max_up" {
  description = "The maximum upload rate (kbps) for the QoS group."
  value       = data.unifi_client_qos_rate.wifi.qos_rate_max_up
}
