# UniFi Port Action Examples

This directory contains examples of using the `unifi_port` action to configure
PoE settings on UniFi switch ports.

## Usage

The `unifi_port` action allows you to configure the PoE mode of a specific port
on a UniFi device. This is useful for dynamically managing power delivery to
connected devices.

## Configuration

The action requires the following parameters:

- `device_mac` (Required) - MAC address of the UniFi device (e.g., switch)
- `port_number` (Required) - Port number to configure (typically starts at 1)
- `poe_mode` (Required) - PoE mode to set for the port

### Valid PoE Modes

- `auto` - Automatic PoE detection and power delivery
- `pasv24` - Passive 24V PoE
- `passthrough` - PoE passthrough mode
- `off` - Disable PoE on the port

## Example Usage via Terraform CLI

Actions are invoked using the Terraform CLI. Here are some examples:

### Enable PoE in auto mode on port 1

```bash
terraform apply -target='action.unifi_port.enable_poe' \
  -var='device_mac=01:23:45:67:89:AB' \
  -var='port_number=1' \
  -var='poe_mode=auto'
```

### Disable PoE on port 2

```bash
terraform apply -target='action.unifi_port.disable_poe' \
  -var='device_mac=01:23:45:67:89:AB' \
  -var='port_number=2' \
  -var='poe_mode=off'
```

### Set passive 24V mode on port 3

```bash
terraform apply -target='action.unifi_port.passive_poe' \
  -var='device_mac=01:23:45:67:89:AB' \
  -var='port_number=3' \
  -var='poe_mode=pasv24'
```

## Notes

- The action updates the port override configuration for the specified device
- If a port override already exists for the port, only the PoE mode is updated
- If no port override exists, a new one is created with just the PoE mode set
- The device must be already adopted and managed by the UniFi controller
- Use the correct MAC address format (colon-separated hex pairs)

## Prerequisites

- UniFi controller must be accessible
- Device must be adopted and online
- Valid credentials configured in the provider
- The device must support PoE on the specified port
