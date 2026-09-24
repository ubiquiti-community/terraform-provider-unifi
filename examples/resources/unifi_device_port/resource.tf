# unifi_device_port manages a single switch port's overrides as its own
# resource, instead of as a port_override block inside unifi_device - do not
# declare both for the same device_mac + index. Only ports the controller
# already lists in the device's port_overrides (i.e. ports that already
# carry at least one override) can be read or imported - a port left
# entirely at its defaults is not returned by the API and has nothing here
# to manage.
resource "unifi_device_port" "example" {
  device_mac = "aa:bb:cc:dd:ee:ff"
  index      = 1

  name                  = "Port 1"
  native_networkconf_id = unifi_network.example.id
  forward               = "customize"
  poe_mode              = "auto"
}
