resource "unifi_network" "lan" {
  name    = "LAN"
  purpose = "corporate"
  subnet  = "10.0.0.1/24"
}

# Block ads and malware on the LAN, force safe search on YouTube.
resource "unifi_content_filter" "lan" {
  name        = "LAN filter"
  network_ids = [unifi_network.lan.id]
  categories  = ["ADVERTISEMENT", "MALWARE", "PHISHING"]
  allow_list  = ["example.com"]
  safe_search = ["YOUTUBE"]
}
