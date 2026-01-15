data "unifi_client_info_list" "all" {
  # Retrieves all active clients from the default site
}

# Get total client count
output "total_clients" {
  value = length(data.unifi_client_info_list.all.clients)
}
