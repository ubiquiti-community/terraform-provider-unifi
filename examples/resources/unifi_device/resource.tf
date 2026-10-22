data "unifi_port_profile" "disabled" {
  # look up the built-in disabled port profile
  name = "Disabled"
}

resource "unifi_port_profile" "poe" {
  name    = "poe"
  forward = "customize"

  native_networkconf_id = var.native_network_id
  tagged_networkconf_ids = [
    var.some_vlan_network_id,
  ]

  poe_mode = "auto"
}

resource "unifi_device" "us_24_poe" {
  # optionally specify MAC address to skip manually importing
  # manual import is the safest way to add a device
  mac = "01:23:45:67:89:AB"

  name = "Switch with POE"

  port_override {
    index           = 1
    name            = "port w/ poe"
    port_profile_id = unifi_port_profile.poe.id
    poe_mode        = "auto" # auto, pasv24, passthrough, off
  }

  port_override {
    index           = 2
    name            = "disabled"
    port_profile_id = data.unifi_port_profile.disabled.id
  }

  # Link aggregation: port 11 is the aggregate lead, bonding member port 12.
  port_override {
    index             = 11
    op_mode           = "aggregate" # switch, mirror, aggregate
    aggregate_members = [12]
  }
}

# Grouped settings: LED, spanning tree and the per-port feature groups are
# nested objects rather than prefixed attributes.
resource "unifi_device" "us_8_60w" {
  mac  = "01:23:45:67:89:AC"
  name = "Rack switch"

  led = {
    override   = "on"
    color      = "#00ff00"
    brightness = 50
  }

  stp = {
    version  = "rstp"
    priority = 4096
  }

  port_override {
    index = 1
    name  = "camera"

    dot1x = {
      ctrl = "force_authorized"
    }
    port_security = {
      enabled     = true
      mac_address = ["aa:bb:cc:dd:ee:01"]
    }
    stormctrl = {
      type = "level"
      bcast = {
        enabled = true
        level   = 50
      }
    }
  }
}

# Etherlighting on switches that support it (e.g. USW Pro Max). Colors per
# link speed or network are site-wide; see unifi_setting's etherlighting block.
resource "unifi_device" "usw_pro_max_24_poe" {
  mac  = "01:23:45:67:89:AD"
  name = "Switch with Etherlighting"

  etherlighting = {
    mode       = "speed"         # speed, network
    brightness = 100             # 1-100
    behavior   = "breath"        # steady, breath
    led_mode   = "etherlighting" # etherlighting, standard
  }
}
