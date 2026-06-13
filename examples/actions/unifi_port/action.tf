# The unifi_port action configures the PoE mode of a single port on a UniFi
# device (typically a switch). It is config-driven: declare an `action` block
# with the desired arguments and either invoke it directly via the CLI
# (`terraform apply -target='action.unifi_port.<name>'`) or trigger it from a
# resource's lifecycle using an `action_trigger` block (Terraform >= 1.14).

# Arguments (all required):
#   device_mac  - MAC address of the device containing the port.
#   port_number - Port number/index on the device (typically starts at 1).
#   poe_mode    - One of: auto, pasv24, passthrough, off.

# Enable PoE in auto mode on port 1 of a switch.
action "unifi_port" "enable_poe_port1" {
  config {
    device_mac  = "01:23:45:67:89:ab"
    port_number = 1
    poe_mode    = "auto"
  }
}

# Disable PoE on port 2 of the same switch.
action "unifi_port" "disable_poe_port2" {
  config {
    device_mac  = "01:23:45:67:89:ab"
    port_number = 2
    poe_mode    = "off"
  }
}

# Deliver passive 24V PoE on port 3 (e.g. for legacy/airMAX devices).
action "unifi_port" "passive_poe_port3" {
  config {
    device_mac  = "01:23:45:67:89:ab"
    port_number = 3
    poe_mode    = "pasv24"
  }
}

# A managed switch whose lifecycle can trigger the actions above. The device
# must already be adopted by the controller before it can be managed here.
resource "unifi_device" "switch" {
  mac  = "01:23:45:67:89:ab"
  name = "office-switch"

  # Trigger port actions from the device lifecycle (Terraform >= 1.14).
  # `events` accepts before_create / after_create / before_update /
  # after_update. Each referenced action runs in order after the device is
  # successfully created, ensuring PoE state is applied on first provisioning.
  lifecycle {
    action_trigger {
      events = [after_create]
      actions = [
        action.unifi_port.enable_poe_port1,
        action.unifi_port.disable_poe_port2,
      ]
    }

    # Re-apply passive PoE whenever the device is updated.
    action_trigger {
      events  = [after_update]
      actions = [action.unifi_port.passive_poe_port3]
    }
  }
}
