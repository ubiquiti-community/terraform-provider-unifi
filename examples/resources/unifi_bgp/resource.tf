resource "unifi_bgp" "default" {
  description      = "BGP"
  enabled          = true
  upload_file_name = "bgp.conf"

  config = <<EOF
frr defaults traditional
log file stdout
router bgp 65000
  bgp ebgp-requires-policy
  bgp router-id 10.0.0.1
  bgp log-neighbor-changes
  bgp graceful-restart
  bgp bestpath as-path multipath-relax
  !
  neighbor CILIUM peer-group
  neighbor CILIUM remote-as 65001
  neighbor CILIUM ebgp-multihop 2
  neighbor CILIUM timers 3 9
  neighbor CILIUM timers connect 5
  neighbor CILIUM soft-reconfiguration inbound
  !
  bgp listen range 10.1.40.0/26 peer-group CILIUM
  bgp listen range fd00:10::/64 peer-group CILIUM
  !
  address-family ipv4 unicast
    redistribute connected
    neighbor CILIUM activate
    neighbor CILIUM route-map CILIUM-IN in
    neighbor CILIUM route-map CILIUM-OUT out
    neighbor CILIUM maximum-prefix 1000
    neighbor CILIUM next-hop-self
  exit-address-family

  address-family ipv6 unicast
    redistribute connected
    neighbor CILIUM activate
    neighbor CILIUM route-map CILIUM-IN-V6 in
    neighbor CILIUM route-map CILIUM-OUT-V6 out
    neighbor CILIUM maximum-prefix 1000
    neighbor CILIUM next-hop-self
  exit-address-family
!
route-map CILIUM-IN permit 10
!
route-map CILIUM-OUT permit 10
!
route-map CILIUM-IN-V6 permit 10
!
route-map CILIUM-OUT-V6 permit 10
!
line vty
!
EOF
}
