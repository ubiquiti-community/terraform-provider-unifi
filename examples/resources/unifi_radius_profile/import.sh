# RADIUS profiles can be imported using the ID, e.g.
terraform import unifi_radius_profile.external 5f8e2a1b3c4d5e6f7a8b9c0d

# Or with an explicit site using the "site:id" format.
terraform import unifi_radius_profile.external default:5f8e2a1b3c4d5e6f7a8b9c0d
