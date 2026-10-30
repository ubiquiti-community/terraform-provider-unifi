# Import ID is "<device_mac>/<index>" - a slash, not a colon, since the MAC
# address itself is colon-separated.
terraform import unifi_device_port.example aa:bb:cc:dd:ee:ff/1
