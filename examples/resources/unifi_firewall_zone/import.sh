# Firewall zones can be imported using the zone ID, or site:id for a non-default site.
terraform import unifi_firewall_zone.dmz 5f3e9b2c4ee8cb0f1f4a1234

# Built-in zones (e.g. Hotspot, Internal) can also be imported by name, which
# resolves the controller-assigned ID for you. Prefix with "site:" for a non-default site.
terraform import unifi_firewall_zone.hotspot name=Hotspot
