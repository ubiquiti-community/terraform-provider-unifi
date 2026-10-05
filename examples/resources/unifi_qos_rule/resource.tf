# Prioritize an application category (DPI category 8: games) for everyone.
resource "unifi_qos_rule" "gaming" {
  name      = "Prioritize Online Gaming"
  objective = "PRIORITIZE"

  destination = {
    matching_target  = "APP_CATEGORY"
    app_category_ids = [8]
  }
}

# Cap one client at 40/10 Mbit/s, weekday evenings only.
resource "unifi_qos_rule" "limit_downloads" {
  name                = "Limit downloads"
  objective           = "LIMIT"
  download_limit_kbps = 40000
  upload_limit_kbps   = 10000

  source = {
    matching_target = "CLIENT"
    client_macs     = ["9c:6b:00:39:f7:a6"]
  }

  schedule = {
    mode             = "EVERY_WEEK"
    repeat_on_days   = ["mon", "tue", "wed", "thu", "fri"]
    time_range_start = "18:00"
    time_range_end   = "23:00"
  }
}
