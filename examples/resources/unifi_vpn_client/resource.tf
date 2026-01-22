resource "unifi_vpn_client" "wireguard_file" {
  name          = "my-wireguard-vpn"
  enabled       = true
  subnet        = "10.0.0.2/24"
  default_route = true
  pull_dns      = false

  wireguard = {
    private_key = "your_base64_private_key_here"
    interface   = "wan"

    configuration = {
      content  = filebase64("${path.module}/wireguard.conf")
      filename = "wireguard.conf"
    }
  }
}

resource "unifi_vpn_client" "wireguard_manual" {
  name          = "my-manual-wireguard"
  enabled       = true
  subnet        = "10.0.1.2/24"
  default_route = false
  pull_dns      = true

  wireguard = {
    private_key = "your_base64_private_key_here"
    interface   = "wan"
    dns_servers = ["8.8.8.8", "8.8.4.4"]

    peer = {
      ip         = "203.0.113.1"
      port       = 51820
      public_key = "your_peer_public_key_here"
    }
  }
}

resource "unifi_vpn_client" "wireguard_with_psk" {
  name          = "secure-wireguard"
  enabled       = true
  subnet        = "10.0.2.2/24"
  default_route = true
  pull_dns      = false

  wireguard = {
    private_key           = "your_base64_private_key_here"
    preshared_key_enabled = true
    preshared_key         = "your_preshared_key_here"
    interface             = "wan"
    dns_servers           = ["8.8.8.8", "8.8.4.4"]

    peer = {
      ip         = "203.0.113.1"
      port       = 51820
      public_key = "your_peer_public_key_here"
    }
  }
}
