# Power supervisors can be imported by the supervised device's MAC address,
terraform import unifi_power_supervisor.ap_lobby 94:2a:6f:d6:ce:fd

# by the controller-assigned id,
terraform import unifi_power_supervisor.ap_lobby 5f3e9b2c4ee8cb0f1f4a1234

# or site:id / site:mac for a non-default site.
terraform import unifi_power_supervisor.ap_lobby my-site:94:2a:6f:d6:ce:fd
