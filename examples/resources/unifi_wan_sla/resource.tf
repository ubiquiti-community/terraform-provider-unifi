# unifi_wan_sla manages a WAN SLA profile (Network 11: Settings > Profiles > WAN SLA): the check that decides whether
# a WAN is usable. WANs reference it with unifi_wan.sla_id; a WAN whose profile is down is taken out of load balancing
# or failover.
#
# Here a server breaches on packet loss, or on latency and jitter together, and the WAN is unusable when most servers
# breach, so one unreachable target can't take a WAN out.
resource "unifi_wan_sla" "standard" {
  name                = "wan-standard"
  monitor_health_type = "MAJORITY"

  monitors = [
    {
      type              = "ICMP"
      target            = "1.1.1.1"
      interval          = 1
      time_period       = 30
      threshold_policy  = "loss-or-all-others"
      loss_threshold    = 10
      latency_threshold = 150
      jitter_threshold  = 30
    },
    {
      type              = "ICMP"
      target            = "9.9.9.9"
      interval          = 1
      time_period       = 30
      threshold_policy  = "loss-or-all-others"
      loss_threshold    = 10
      latency_threshold = 150
      jitter_threshold  = 30
    },
    {
      type              = "DNS"
      target            = "8.8.8.8"
      domain            = "example.com"
      interval          = 5
      time_period       = 60
      threshold_policy  = "loss-or-all-others"
      loss_threshold    = 10
      latency_threshold = 150
      jitter_threshold  = 30
    },
  ]
}

# Two WANs load balanced 50/50, both judged by the profile.
resource "unifi_wan" "primary" {
  name         = "Internet 1"
  load_balance = { type = "weighted", weight = 50, failover_priority = 1 }
  sla_id       = unifi_wan_sla.standard.id
}

resource "unifi_wan" "secondary" {
  name         = "Internet 2"
  load_balance = { type = "weighted", weight = 50, failover_priority = 2 }
  sla_id       = unifi_wan_sla.standard.id
}
