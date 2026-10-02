# unifi_dns_record manages local DNS records served by the UniFi gateway.
# record_type must be one of: A, AAAA, CNAME, MX, TXT, SRV, PTR, NS.

# A record: maps a hostname to an IPv4 address.
resource "unifi_dns_record" "host_a" {
  name        = "nas.example.com"
  record_type = "A"
  value       = "192.168.1.100"
  ttl         = "5m"
}

# AAAA record: maps a hostname to an IPv6 address.
resource "unifi_dns_record" "host_aaaa" {
  name        = "nas.example.com"
  record_type = "AAAA"
  value       = "2001:db8::100"
  ttl         = "5m"
}

# CNAME record: aliases one name to another.
resource "unifi_dns_record" "www_cname" {
  name        = "www.example.com"
  record_type = "CNAME"
  value       = "nas.example.com"
  ttl         = "1h"
}

# TXT record: arbitrary text, e.g. an SPF policy.
resource "unifi_dns_record" "spf_txt" {
  name        = "example.com"
  record_type = "TXT"
  value       = "v=spf1 mx -all"
  ttl         = "1h"
}

# MX record: mail exchanger with a priority.
resource "unifi_dns_record" "mail_mx" {
  name        = "example.com"
  record_type = "MX"
  value       = "mail.example.com"
  priority    = 10
  ttl         = "1h"
}

# SRV record: service location with priority, weight, and port.
resource "unifi_dns_record" "sip_srv" {
  name        = "_sip._tcp.example.com"
  record_type = "SRV"
  value       = "sip.example.com"
  priority    = 10
  weight      = 60
  port        = 5060
  ttl         = "1h"
}

# NS record: delegates a domain or subdomain to a name server.
resource "unifi_dns_record" "forward_ns" {
  name        = "nas.example.com"
  record_type = "NS"
  value       = "ns1.example.com"
}