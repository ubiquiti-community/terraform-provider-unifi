# Import ID is "<device_mac>/<index>", or "<site>/<device_mac>/<index>" for a
# site other than the provider's. The separator is a slash because the MAC
# address itself is colon-separated.
terraform import unifi_device_port.nas aa:bb:cc:dd:ee:ff/8
