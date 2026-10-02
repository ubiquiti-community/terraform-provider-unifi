# Import using the site's 24-hex controller id (the `_id` from
# /proxy/network/api/self/sites).
terraform import unifi_site.mysite 5fe6261995fe130013456a36

# Or import by the site's short name. Any value that is not a 24-hex id is
# treated as a name; use the `name=` prefix to be explicit (e.g. the default
# site is named `default`).
terraform import unifi_site.mysite name=default

# Note: the UUID shown in the UI / Integration API (.../integration/v1/sites)
# is NOT the import id — use the 24-hex `_id` or the site name above.
