# Look up a DNS record by name (hostname/key).
data "unifi_dns_record" "example" {
  name = "nas.example.com"
}

# The record type (e.g. A, AAAA, CNAME) and its value (computed).
output "dns_record_type" {
  value = data.unifi_dns_record.example.type
}

output "dns_record_value" {
  value = data.unifi_dns_record.example.value
}

# The TTL and whether the record is enabled (computed).
output "dns_record_ttl" {
  value = data.unifi_dns_record.example.ttl
}

output "dns_record_enabled" {
  value = data.unifi_dns_record.example.enabled
}

# Look up a record on a specific site.
data "unifi_dns_record" "site_specific" {
  site = "production"
  name = "printer.example.com"
}
