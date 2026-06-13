# A VPN server is configured with exactly one of the `wireguard`, `l2tp`, or
# `openvpn` nested blocks. The block you choose determines the server type.
# The `subnet` is the server's tunnel network in gateway form (the first
# address, e.g. `10.100.0.1/24`, is the server's tunnel IP).

# WireGuard server. If `private_key` is omitted the provider generates one at
# create time and exposes the derived `public_key` as a computed attribute
# (useful for configuring `unifi_wireguard_peer` resources).
resource "unifi_vpn_server" "wireguard" {
  name   = "wireguard"
  subnet = "10.100.0.1/24"

  wireguard = {
    port = 51820
  }
}

# WireGuard server that also pushes custom DNS servers to connecting clients and
# binds to a specific WAN interface. `dns.enabled` defaults to true when
# `servers` is non-empty.
resource "unifi_vpn_server" "wireguard_dns" {
  name   = "wireguard-dns"
  subnet = "10.101.0.1/24"

  wireguard = {
    port = 51821
  }

  dns = {
    servers = ["1.1.1.1", "1.0.0.1"]
  }

  wan = {
    ip        = "any"
    interface = "wan"
  }
}

# L2TP/IPsec server. `pre_shared_key` is required by the controller. L2TP
# servers authenticate users against a RADIUS profile.
resource "unifi_radius_profile" "l2tp" {
  name = "l2tp-radius"

  auth_server {
    ip     = "192.168.1.100"
    port   = 1812
    secret = "radius-secret"
  }
}

resource "unifi_vpn_server" "l2tp" {
  name             = "l2tp"
  subnet           = "10.110.0.1/24"
  radiusprofile_id = unifi_radius_profile.l2tp.id

  l2tp = {
    pre_shared_key     = "change-me-to-a-strong-secret"
    allow_weak_ciphers = false
  }
}

# OpenVPN server. The controller generates the certificates and keys, which are
# exposed as computed (sensitive) attributes. `mode` is one of `server` or
# `site-to-site`; `encryption_cipher` is one of `AES_256_GCM`, `AES_256_CBC`,
# or `BF_CBC`.
resource "unifi_vpn_server" "openvpn" {
  name             = "openvpn"
  subnet           = "10.120.0.1/24"
  radiusprofile_id = unifi_radius_profile.l2tp.id

  openvpn = {
    port              = 1194
    mode              = "server"
    encryption_cipher = "AES_256_GCM"
  }
}
