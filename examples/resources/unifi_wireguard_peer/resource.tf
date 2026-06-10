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
