resource "unifi_dns_record" "test" {
  name        = "test-record.example.com"
  enabled     = true
  priority    = 10
  record_type = "A"
  ttl         = 300
  value       = "192.168.1.100"
}
