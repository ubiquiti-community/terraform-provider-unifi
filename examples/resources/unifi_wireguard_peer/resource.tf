resource "unifi_vpn_server" "wg" {
  name   = "wireguard"
  subnet = "192.0.2.1/24"

  wireguard = {
    port = 51820
  }
}

resource "unifi_wireguard_peer" "example" {
  network_id   = unifi_vpn_server.wg.id
  name         = "example-peer"
  interface_ip = "192.0.2.10"
  public_key   = "ZmFrZS10ZXN0LXdpcmVndWFyZC1wdWJrZXkAAAAAAAA="
}

# A peer with a pre-shared key. Prefer preshared_key_wo: it is used at apply
# time but never written to state, at the cost of the provider being unable to
# detect a key changed outside Terraform. Leaving both unset keeps the key off
# the wire entirely, so a key configured in the UI is preserved.
resource "unifi_wireguard_peer" "with_psk" {
  network_id   = unifi_vpn_server.wg.id
  name         = "laptop"
  interface_ip = "192.0.2.11"
  public_key   = "ZmFrZS10ZXN0LXdpcmVndWFyZC1wdWJrZXkAAAAAAAA="

  preshared_key_wo = var.laptop_psk
}
