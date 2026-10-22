# unifi_traffic_matching_list manages a reusable set of match criteria (IPv4
# addresses or ports) exposed by the UniFi Network Integration API and referenced
# by firewall and traffic-management rules.
#
# NOTE: The UniFi Integration API addresses sites by UUID. The "site" attribute
# (which defaults to the provider's configured site) accepts either a site name
# such as "default" or a site UUID; a name is resolved to its UUID automatically.

# A list of IPv4 addresses, e.g. upstream DNS resolvers.
resource "unifi_traffic_matching_list" "dns_resolvers" {
  name = "DNS Resolvers"
  type = "IPV4_ADDRESSES"

  items = [
    "192.0.2.4",
    "198.51.100.53",
  ]
}

# A list of ports. For type = PORTS the items are port numbers given as strings.
resource "unifi_traffic_matching_list" "web_ports" {
  name = "HTTP(S)"
  type = "PORTS"

  items = [
    "80",
    "443",
  ]
}
