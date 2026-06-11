# Manual site-to-site IPsec VPN to a remote gateway.
resource "unifi_site_to_site_vpn" "hq_to_branch" {
  name           = "HQ-to-Branch"
  peer_ip        = "203.0.113.10"
  pre_shared_key = var.s2s_pre_shared_key
  remote_subnets = ["192.168.20.0/24"]
}

# Write-only pre-shared key sourced from an ephemeral resource (Terraform 1.11+),
# with a customized IKE/ESP proposal.
resource "unifi_site_to_site_vpn" "branch" {
  name              = "Branch-Office"
  interface         = "wan"
  peer_ip           = "203.0.113.20"
  key_exchange      = "ikev2"
  pre_shared_key_wo = ephemeral.vault_kv_secret_v2.s2s.data["psk"]
  remote_subnets    = ["10.10.0.0/16"]

  profile        = "customized"
  ike_encryption = "aes256"
  ike_hash       = "sha256"
  ike_dh_group   = 14
  esp_encryption = "aes256"
  esp_hash       = "sha256"
  esp_dh_group   = 14
  pfs            = true
}
