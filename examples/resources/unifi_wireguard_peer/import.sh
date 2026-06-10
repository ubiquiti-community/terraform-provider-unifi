# import by network ID and peer ID from provider configured site
terraform import unifi_wireguard_peer.laptop 5dc28e5e9106d105bdc87217:6a259bc465f49edd8b44d2f4

# import from another site
terraform import unifi_wireguard_peer.laptop bfa2l6i7:5dc28e5e9106d105bdc87217:6a259bc465f49edd8b44d2f4
