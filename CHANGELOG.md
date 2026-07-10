# Changelog

All notable changes to this project will be documented in this file.

## [v0.55.0] - 2026-07-10

### ✨ Features

- **`unifi_ap_group`: manage AP group membership.** Full CRUD, complementing the existing read-only data source. Which APs belong to a group was fixed in the controller UI: the data source could read a group, but nothing could create or edit one, so `unifi_wlan.ap_group_ids` could only reference groups built by hand. The resource writes membership through the v2 `apgroups` API. `device_macs` reuses the `unifi_client` MAC type, so `AA-BB-…` and `aa:bb:…` read back equal rather than churning the plan on every refresh. Import takes the group ID, or `site:id` for a non-default site (#359, go-unifi#52).

### 🐛 Bug Fixes

- **`unifi_firewall_policy`: fix creating a policy that matches an IP group failing with `api.err.EmptyFirewallDestinationIps` (400).** A `source`/`destination` referencing an address group via `ip_group_id` (#316) must be sent with `matching_target_type = "OBJECT"`, but the #293 derivation back-filled an empty type as `SPECIFIC` for any non-ANY match — and on create the type is never controller-assigned, so every create with `ip_group_id` was rejected and only literal `ips` worked. A group reference now derives `OBJECT`, also overriding a stale `""`/`"ANY"`/`"SPECIFIC"` carried in state so switching an existing policy from literal `ips` to a group reference works on update too; a controller-assigned `OBJECT`/`LIST` is still preserved (#365, #316, #293)
- **`unifi_device`: carry `switch_vlan_enabled`, `radio_table[].vwire_enabled`, and `mesh_sta_vap_enabled` in the update PUT.** The update PUT was assembled from a minimal `Device` that dropped several configured fields, so the controller never received them and every apply that set one failed with `inconsistent result after apply` (`was cty.True, but now cty.False`). Fixed for: `switch_vlan_enabled` (the "Port VLAN" toggle, e.g. an AP with a built-in switch, where the toggle is what makes VLAN tagging take effect on the built-in ports); `radio_table[].vwire_enabled` (the "Mesh Parent" toggle — the whole `radio_table` was omitted from the minimal PUT, dropping every radio sub-field); and `mesh_sta_vap_enabled` (the "Mesh Connect" toggle, newly added to the `unifi_device` schema as an `Optional + Computed` bool). All are now carried in the PUT body when configured. `omitempty` (at every level, including `radio_table`) keeps a `false`/empty off the wire, so it never disturbs the controller default. Verified against a real controller (#363)
- **`unifi_device` / `unifi_setting`: stop controller-managed lists churning to "known after apply" on unrelated edits.** Several `Optional + Computed` lists were replanned as `(known after apply)` whenever any other field on the same resource changed — a spurious diff (the same class as #338). They now use `UseStateForUnknown`, keeping their prior value unless explicitly changed: `unifi_device` `radio_table` and `outlet_overrides`, and `unifi_setting` `contents` (syslog facilities), `server_names` (DoH), `enabled_categories` / `enabled_networks` (IPS), and `network_ids` (IGMP snooping).
- **`unifi_ap_group`: allow empty membership and stop empty groups reading back as `null`.** `device_macs` was `Required` with a `SizeAtLeast(1)` validator, and the read mapped an empty member list to `SetNull` — so a group the controller legitimately allows to have zero members (the API returns 201 for an empty membership) could not be authored, and importing one surfaced as an empty-vs-`null` inconsistency. `device_macs` now accepts an empty set and reads empty back as an empty set. The built-in default "All APs" group (which the controller marks read-only) is documented as non-editable through the resource.

## [v0.54.1] - 2026-07-05

### 🐛 Bug Fixes

- **`unifi_radius_profile`: make `auth_server` / `acct_server` `ip` optional so the default profile can be imported.** The controller-managed default RADIUS profile (created when a gateway RADIUS/VPN service is enabled, with `use_usg_auth_server = true`) returns a server entry without an IP. `ip` was `Required`, so re-declaring an imported profile failed with `The argument "ip" is required`, and an empty IP read back as `""` instead of null. `ip` is now `Optional` and an absent IP maps to null, so the default profile round-trips cleanly (#356)

## [v0.54.0] - 2026-07-02

### ✨ Features

- **`unifi_network`: expose the network `purpose` (`corporate`, `guest`, `vlan-only`).** A new `Optional + Computed` attribute. The provider previously hard-coded `corporate` (or `vlan-only` for a `third_party_gateway`), so a `guest` network could not be authored and a controller-assigned guest purpose was silently fought on every apply. `purpose` is now sent when configured and read back from the controller. **Note:** on Zone-Based-Firewall controllers the purpose is coupled to the firewall zone — a `guest` network only keeps `purpose = "guest"` while it belongs to the guest/Hotspot zone (assign it there via `unifi_firewall_zone`); placed in a non-guest zone the controller rewrites it back to `corporate`. `third_party_gateway = true` still forces `vlan-only` for backward compatibility (#276)
- **`unifi_firewall_policy`: make `connection_state_type` / `connection_states` author-settable.** Both attributes were `Computed`-only, so setting them returned `Invalid Configuration for Read-Only Attribute` — you could not author a policy scoped to a specific connection state. They are now `Optional + Computed`: leave them unset and the controller manages them as before, or set `connection_state_type = "CUSTOM"` with `connection_states = ["NEW", …]` (or `RESPOND_ONLY`) to author, for example, a `NEW`-only logging/deny policy that coexists with stateful returns in a zone-based firewall. Values are validated (`ALL`/`RESPOND_ONLY`/`CUSTOM`; states `NEW`/`ESTABLISHED`/`RELATED`/`INVALID`) and still round-trip on update (#351)
- **`unifi_wan`: expose `networkgroup` (`WAN`, `WAN2`, …).** A new computed-by-default attribute identifying which WAN group an interface belongs to. The provider previously hard-coded `wan_networkgroup`/`attr_hidden_id` to `WAN`, so updating a **secondary** uplink (`WAN2`) collided with the primary and the controller rejected the PUT (`api.err.WanConfigurationForNetworkGroupAlreadyExists`). The group is now read from the controller and preserved in the update payload (`UseStateForUnknown`, so an imported `WAN2` needs no explicit config), making multi-WAN setups manageable (#334)

### 🐛 Bug Fixes

- **`unifi_network`: fix `inconsistent result after apply` on `multicast_dns` for non-vlan-only networks.** Some controllers (notably UniFi OS gateways) ignore the per-network `mdns_enabled` flag and always store `false`, so a configured `true` conflicted with the post-apply read. The corporate-network read path now preserves the configured value (the vlan-only path already did), falling back to the controller's value only when it was left unset (#282)
- **`unifi_wan`: fix `inconsistent result after apply` on `wan_dslite_remote_host_auto`.** The controller can force this field back to `true` server-side, so the post-apply read conflicted with a user-configured `false`. The create/update paths now re-assert the configured DS-Lite values on the post-apply state (the update path applied its write-preserve before the API round-trip, so the controller value won); the next refresh still reconciles with the controller (#281)
- **`unifi_firewall_policy`: stop `source`/`destination` match lists churning to "known after apply" on unrelated edits.** Changing any other field (e.g. `index` or `protocol`) replanned `network_ids`, `client_macs`, `ips` and `web_domains` as `(known after apply)` — showing a spurious diff and risking the controller recomputing them. These Computed attributes now use `UseStateForUnknown`, so they keep their prior value unless explicitly changed (#338)
- **`unifi_device`: fix LED updates failing with `inconsistent result after apply`.** The update PUT body was assembled as a minimal device that dropped the LED override fields (`led_override`, `led_override_color`, `led_override_color_brightness`), so the controller kept the old values and the post-apply read conflicted with the plan. They are now included in the PUT, and — because the controller applies LED changes to APs asynchronously — the update path also re-asserts the planned LED values on the post-apply state, leaving the next refresh to reconcile with the controller (#337)
- **`unifi_device`: fix `mgmt_network_id` (Network Override) never persisting.** The update PUT was assembled from a minimal `Device` that dropped a configured `mgmt_network_id`, so the controller never received it: every apply that set it failed with `inconsistent result after apply`, and the per-device management VLAN could not be set through the provider. The field is now carried in the PUT body when configured. `omitempty` keeps a null value off the wire, so it never reintroduces the #177 zero-value rejection. The tag-upstream-first connectivity requirement documented in #330 still applies (#329)
- **`unifi_wan`: fix `inconsistent result after apply` on `dns` address fields (`primary`, `secondary`, `ipv6_primary`, `ipv6_secondary`).** When no DNS server is configured the controller persists and returns an empty string `""`, but these Optional fields plan as `null`, so the post-apply read conflicted with the plan (e.g. after import with IPv6 DNS preference `auto`). The read now normalizes `""` (and a nil pointer) to `null`, so unset addresses stay null and a real address still round-trips (#333)
- **`unifi_firewall_policy`: make `index` read-only to stop `inconsistent result after apply` and a perpetual diff.** Pinning `index` failed: the controller ignores a client-supplied value and always appends the policy at the end of its source/destination zone-pair, so the post-apply read (e.g. `10010` → `10020`) conflicted with the plan and then looped forever. Verified against a real UniFi OS 10.x controller — the supported integration API rejects `index` as input and exposes no reorder operation, so policy ordering cannot be managed through the provider. `index` is now `Computed` (controller-assigned) and the provider no longer sends it; reorder policies in the UniFi UI if needed (#348)

## [v0.53.0] - 2026-06-24

### ✨ Features

- **`unifi_firewall_policy`: match an IP group on `source`/`destination`.** A new `ip_group_id` attribute references a `unifi_firewall_group` of type address-group (used with `matching_target = "IP"` and `matching_target_type = "OBJECT"`), alongside the existing `port_group_id`. Backed by a go-unifi change adding the `ip_group_id` field to the firewall-policy source/destination structs (#316)
- **`unifi_dns_record`: support `NS` records.** `record_type` now accepts `NS`, enabling Forward Domain entries (delegating a domain to another name server). Schema, validator, docs and an example were updated (#318, #319)

### 🐛 Bug Fixes

- **`unifi_firewall_policy`: fix `inconsistent result after apply` on `source`/`destination` `matching_target_type` when updating a policy (e.g. changing `action`).** This field is firmware-derived: the controller (and the provider's own derivation for #293) may set it to a concrete value during the update PUT (e.g. `""` → `"SPECIFIC"` for a non-ANY match), which the planned value cannot anticipate when the prior state still carries an empty type. The update path now re-asserts the planned value on the post-apply state, leaving the next refresh to reconcile it with the controller (#324)
- **`unifi_wlan`: fix `inconsistent result after apply` on controller-managed fields.** `minimum_data_rate_2g_kbps`/`minimum_data_rate_5g_kbps` defaulted to `0`, but the controller assigns its own value in `auto` mode (e.g. `1000`/`6000`); they are now `Computed` (via `UseStateForUnknown`) instead of statically defaulted. `radius_profile_id` and `bc_filter_list` were `Optional`-only yet the controller populates them on its own, so they too became `Optional + Computed`. When these are left unset, the controller's value is now accepted instead of conflicting with a `0`/`null` plan (#323)

### 📚 Documentation

- **`unifi_device`: document the `mgmt_network_id` tag-upstream-first requirement.** Setting the Network Override tags the device's management onto the target VLAN; if that VLAN is not tagged on the device's upstream port the device drops off and the apply fails with an inconsistent-result error. The description now spells out the two-step apply (tag the uplink first, then set `mgmt_network_id`) (#329, #330)

## [v0.52.4] - 2026-06-17

### 🐛 Bug Fixes

- **`unifi_firewall_zone`: create no longer fails with "Unrecognized field default_zone" (400) on UniFi Network 10.4.x.** The server-computed `default_zone` was always serialized into the create request; it is now omitted (modeled as `*bool` in go-unifi) and only read back as a computed attribute (#310)

### 📚 Documentation

- **`unifi_network`: clarify that `subnet` sets the gateway IP.** A custom gateway is already supported — the host portion of `subnet` is the gateway (e.g. `10.0.10.254/24` → gateway `.254`); it need not be the first usable address (#308, #309)

## [v0.52.3] - 2026-06-17

### 🐛 Bug Fixes

- Fix operation timeouts for the list resources, and add acceptance tests for them

## [v0.52.2] - 2026-06-16

### 🐛 Bug Fixes

- **`unifi_firewall_policy`: set `matching_target_type` for specific matches.** Switching a `source`/`destination` from `matching_target = "ANY"` to a specific target (e.g. `"IP"`) left `matching_target_type` empty, so the update was rejected with `api.err.MissingFirewallPolicySourceMatchingTargetType (400)`. The provider now sends `SPECIFIC` for a non-ANY match (preserving a controller-assigned `OBJECT`/`LIST`) (#293)

---

## [v0.52.1] - 2026-06-16

### 🐛 Bug Fixes

- **`unifi_setting`: stop serializing `0` for unset numeric fields.** An unset Optional+Computed integer was sent as `0`, which the controller rejects (e.g. syslog `netconsole_port: 0` → `400 api.err.InvalidPayload`). The provider now omits `syslog.port`/`syslog.netconsole_port`, `lcm.brightness`/`lcm.idle_timeout`, and `ips` alert `gid`/`id` when unset (#303)

---

## [v0.52.0] - 2026-06-16

### ✨ Features

- **`unifi_setting` `mgmt` block — full management settings** (#274): `advanced_feature_enabled`, `auto_upgrade_hour`, `debug_tools_enabled`, `direct_connect_enabled`, `unifi_idp_enabled`, `wifiman_enabled`, `ssh_username`, `ssh_password` (sensitive), `ssh_auth_password_enabled`. Configured fields are overlaid onto the controller's current settings, so unset fields are preserved.
- **`unifi_setting` `ips` block — signature alert suppression** (#275): new `suppression_alerts` list (`category`, `gid`, `id`, `signature`, `type`) with a nested `tracking` list (`direction`, `mode`, `value`).

---

## [v0.51.0] - 2026-06-16

### ✨ Features

- **`unifi_client`: new read-only `last_ip` attribute** — the most recent IP the controller has seen for the client (#287)
- **`unifi_setting`: new `auto_speedtest` block** — periodic internet speed test (`enabled`, `cron_expr`) (#272)
- **`unifi_setting`: six more setting categories** (#273):
  - `dpi` — Deep Packet Inspection (`enabled`, `fingerprinting_enabled`)
  - `lcm` — device display (`enabled`, `brightness`, `idle_timeout`, `sync`, `touch_event`)
  - `network_optimization` — automated network optimization (`enabled`)
  - `ntp` — time servers (`ntp_server_1..4`, `setting_preference`)
  - `syslog` — remote rsyslog (`enabled`, `ip`, `port`, `contents`, `log_all_contents`, `debug`, `this_controller`/`this_controller_encrypted_only`, `netconsole_*`)
  - `country` — regulatory `code`

---

## [v0.50.0] - 2026-06-16

### ⚠️ Breaking Changes

- **`unifi_firewall_policy` `source.port`/`destination.port` are now strings** (were numbers). Update configs from `port = 161` to `port = "161"`. Existing state is migrated automatically by a schema upgrader, so no manual action is required. This is what fixes #288 below and adds comma-separated port lists (#286).

### ✨ Features

- **List resources for 19 more managed resources** (5 → 24 listable), enabling `terraform query` / config-driven import workflows: `radius_user`, `dns_record`, `dynamic_dns`, `radius_profile`, `firewall_group`, `port_forward`, `static_route`, `traffic_route`, `wan`, `vpn_client`, `vpn_server`, `wireguard_peer`, `device`, `client_qos_rate`, `site`, `power_supervisor`, `firewall_rule`, `network`, `port_profile` (#277, #279)
- **Per-resource operation timeouts** — resources and data sources now accept a standardized `timeouts` block (create/read/update/delete) (#285)
- **`unifi_firewall_policy` ports accept a comma-separated list** (e.g. `"80,443"`) and round-trip correctly on import (#286)

### 🐛 Bug Fixes

- **`unifi_firewall_policy`: a portless source/destination no longer freezes the gateway firewall.** A policy with `port_matching_type = ANY` was serialized with `port = "0"`, which current UniFi OS rejects (valid ports are 1–65535) — silently dropping the *entire* firewall ruleset while `apply` reported success. Portless endpoints now omit the port field entirely (#288)
- **`unifi_wlan`: `enhanced_iot = true` no longer fails with "provider produced inconsistent result after apply".** When enhanced IoT is enabled the controller forces `iapp_enabled`, `wpa3_support`, `wpa3_transition`, `pmf_mode` and `dtim_ng`; the provider now pins those fields to the controller's values so apply and subsequent plans stay consistent (#283)

### 🔧 Maintenance

- CI: gate `golangci-lint` on newly-introduced issues only, so a `latest`-tracking linter no longer blocks every PR on pre-existing findings, and clear the existing findings in the test suite (#294)
- CI: workflow cleanup, coverage reporting, and stricter dependency linting (#278, #285)
- Build(deps): bump `golangci/golangci-lint-action` 8 → 9.2.1 (#291) and `codecov/codecov-action` 5 → 7 (#289)

---

## [v0.49.0] - 2026-06-12

### ✨ Features

- **New `unifi_power_supervisor` resource — UniFi Device Supervisor** (UniFi Network 10.2+). Watch a device's heartbeat and have the controller automatically power-cycle its upstream PoE source after a silence threshold. Reference the supervised device by `device_mac`; set the `heartbeat_interval` / `silence_threshold` / `power_off_duration` timings (seconds). The controller resolves the upstream PoE port automatically (`power_sources` is computed). Full CRUD + import by `id`, `site:id`, or the device's MAC. Backed by a new go-unifi v2 client; live-validated on UniFi Network 10.4.57. Note: the supervised device must be powered by a controller-manageable PoE port — a non-PoE uplink is rejected with `PORT_NOT_POE_CAPABLE` (#244)

### 🐛 Bug Fixes

- **Surface the controller's actual error message on v2 API failures.** Errors from the v2 API (firewall policy/zone, wireguard peer, power supervisor) previously showed only a bare `(400 Bad Request)` because the SDK parsed only the v1 error shape. The underlying go-unifi SDK now reads the v2 error body too, so failures include the controller's reason and code (e.g. `api.err.PurePoeRequiresUplinkException: … PORT_NOT_POE_CAPABLE`)

---

## [v0.48.0] - 2026-06-12

### ✨ Features

- **`unifi_firewall_policy`: allow `protocol = "icmp"` / `"icmpv6"`.** The protocol validator only accepted `all`/`tcp`/`udp`/`tcp_udp`, so zone-based firewall ICMP policies could not be planned even though the controller (UniFi Network 10.4.57) accepts and returns them. The firmware-managed `icmp_typename` / `icmp_v6_typename` fields are already round-tripped, so the validator was the only blocker. Note: the controller rejects `create_allow_respond = true` for ICMP policies (`FirewallPolicyCreateRespondTrafficPolicyNotAllowed`) — keep it `false` and add an explicit reverse policy for the reply (#259)

### 🐛 Bug Fixes

- **`unifi_device`: stop a single `port_override` from wiping every other port.** The UniFi `PUT /rest/device/<id>` treats `port_overrides` as a full-replace array, and the provider sent only the declared subset — so declaring one port silently dropped all other ports' overrides back to the default VLAN (a port carrying e.g. an NVR on a CCTV VLAN would lose connectivity). The provider now merges the declared `port_override` blocks (by `index`) onto the device's current overrides before the PUT, making `port_override` **partial management**: manage only the ports you declare, leave the rest untouched. Removing a block stops managing that port but does not reset it (#266)

---

## [v0.47.2] - 2026-06-12

### 🐛 Bug Fixes

- **`unifi_site`: fix provider panic when importing/reading with an unmatched identifier.** Importing a site by an identifier that is neither a 24-hex controller id nor a known site name (e.g. the UUID shown in the UI / Integration API) crashed the provider with a nil-pointer dereference. The read paths now return cleanly on not-found, and `siteToModel` guards against a nil site. Import docs clarify the supported forms (24-hex `_id` or `name=<site-name>`) (#261)
- **`unifi_wan`: fix spurious plan diff after import.** Two read quirks made an imported WAN unable to reach `No changes` without an apply: `vlan.id` was read as null (so it always wanted `+ id = 0`) and is now mapped to the schema default `0`; and `provider_capabilities` (the detected line rate) became `Optional + Computed` with `UseStateForUnknown`, so omitting it from config no longer tries to clear it (#262)

---

## [v0.47.1] - 2026-06-11

### 🔒 Security

- **Stop leaking secrets in error messages.** A failed create/update embedded the raw request payload in the error — including `x_wireguard_private_key`, `x_passphrase`, and `x_ipsec_pre_shared_key` in cleartext — exposing them in terminal scrollback and CI logs. The underlying go-unifi SDK now redacts sensitive fields from payloads in error messages (#256)

### 🐛 Bug Fixes

- **`unifi_vpn_server`: generate the WireGuard `private_key` when unset.** The controller does not generate one (it rejects creation with `api.err.WireguardMissingPrivateKey`) despite the schema marking the field optional/computed. The provider now generates a valid key at create time, and the subnet docs note that the **gateway** form (`10.x.0.1/24`) is required, not the network address (#255)

- **`unifi_network`: fix `inconsistent result after apply` / perpetual diffs on the IPv6 RA/PD attributes.** Networks that carry controller-set RA/PD values (`ipv6_ra`, `ipv6_ra_priority`, `ipv6_ra_preferred_lifetime`, `ipv6_ra_valid_lifetime`, `ipv6_pd_start`, `ipv6_pd_stop`, `ipv6_pd_auto_prefixid_enabled`) — common even on v4-only networks — drifted forever (e.g. `ipv6_ra: true -> false`, `ipv6_pd_start: "::2" -> null`) and could fail apply. These are now `Optional + Computed` with `UseStateForUnknown`, and unset values are no longer serialized as `""`/`0`, so controller-normalized values are preserved instead of clobbered. Extends the v0.47.0 fix to `unifi_network` (#253)
- **`unifi_network`: fix create failing with `api.err.InvalidPayload` when `ipv6_client_address_assignment` is unset.** The attribute (added in v0.45.0) is `Optional + Computed`, so on create it was serialized as an empty string `""`, which the controller rejects — breaking network creation unless the field was pinned to a value. It is now omitted from the payload when unset (#252)
- **`unifi_wan`: allow `type_v6 = "slaac"`.** The validator only accepted `dhcpv6`/`static`/`disabled`, but the controller also supports `slaac` — and **requires** it when the IPv6 delegation type is `single_network` (`api.err.SingleNetworkMustBeSLAAC` otherwise). This blocked enabling IPv6 on the WAN for ISPs that deliver it by Router Advertisement (e.g. Free/Freebox in bridge mode). Validated live on UniFi Network 10.4.57 (#250)

---

## [v0.47.0] - 2026-06-11

### ✨ Features

- **`unifi_firewall_policy`: match traffic by domain/FQDN.** A new `web_domains` attribute on `source` and `destination` (used with `matching_target = "WEB"`) lets a policy filter on hostnames. Backed by a go-unifi change that adds the `web_domains` field and the `WEB` matching target to the firewall-policy schema (#242)

### 🐛 Bug Fixes

- **`unifi_firewall_policy`: actually send/read `network_ids` and `client_macs`.** These match fields were exposed in the schema but never wired to the API — the provider dropped them on write and forced them to `null` on read. They now round-trip like `ips` (#242)
- **`unifi_device`: fix `Provider produced inconsistent result after apply` that broke every device update.** Write-only attributes never returned by the controller (`forget_on_destroy`, `allow_adoption`) are no longer clobbered to `null` by prior state (notably after an import), and the LED attributes (`led_override`, `led_override_color`, `led_override_color_brightness`) now preserve their configured value when the controller does not echo them back. All five gained `UseStateForUnknown` plan modifiers (#243)
- **`unifi_port_profile`: fix `inconsistent result after apply` on `stp_port_mode` and `excluded_networkconf_ids`.** `stp_port_mode` is now actually round-tripped to/from the controller (it was forced to `null` and never sent), and both attributes became `Optional + Computed` with `UseStateForUnknown` so controller-computed values no longer conflict with the plan (#245)
- **`unifi_wlan`: fix `inconsistent result after apply` on `dtim_ng`/`dtim_na`/`dtim_6e` and `iapp_enabled`.** The DTIM fields became `Optional + Computed` so controller defaults (e.g. `1`/`3`/`3`) are accepted when unset, and `iapp_enabled` dropped its static `false` default (the controller may return `true`) in favor of `Optional + Computed` + `UseStateForUnknown` (#245)

---

## [v0.46.0] - 2026-06-11

### ✨ Features

- **New `unifi_site_to_site_vpn` resource** — manage a UniFi manual site-to-site IPsec VPN (`purpose = site-vpn`, `vpn_type = ipsec-vpn`). Exposes the tunnel essentials (`peer_ip`, `interface`, `key_exchange`, `remote_subnets`, `pre_shared_key`) plus the full `profile = customized` IKE/ESP tuning surface (encryption, hash, DH groups, lifetimes, PFS, dynamic routing, route distance). The pre-shared key supports a write-only variant (`pre_shared_key_wo`, Terraform 1.11+). Backed by a go-unifi fix that completes the previously-stubbed site-VPN marshaler. Validated live on UniFi Network 10.4.57 (#78, #239)

### 🧹 Maintenance

- Added a regression unit test for the `unifi_device` `port_override` refresh crash fixed in v0.45.1, and removed a duplicate initialization left by merging the parallel fix (#236, #240)

---

## [v0.45.1] - 2026-06-11

### 🐛 Bug Fixes

- `unifi_device`: fix a refresh/plan crash (`Value Conversion Error … types.ListType[!!! MISSING TYPE !!!]` on `tagged_networkconf_ids`) that hit any device with `port_override` blocks. The override read path now initializes the list to a typed null. Note: `tagged_networkconf_ids` is not yet round-tripped (it reads as null) pending the field being added to the go-unifi SDK (#235, #237)

---

## [v0.45.0] - 2026-06-10

### ✨ Features

- **`unifi_network.ipv6_client_address_assignment`** — new optional+computed attribute to declaratively pin how clients on a network obtain an IPv6 address: `slaac` (SLAAC only), `dhcpv6` (DHCPv6 only), or `slaac-dhcpv6` (both). UI: Networks → IPv6 → Client Address Assignment. Backed by a go-unifi fix that emits the field in the corporate/guest marshalers (it was decode-only before). Validated on a live UniFi Network 10.4.57 controller (#232, #233)

### 🐛 Bug Fixes

- **Login rate-limit resilience** — username/password auth no longer fails with `Unable to Create HTTP Client` when several back-to-back operations (`init → import → plan → plan → apply`) exhaust the controller's `POST /api/auth/login` rate-limit. The SDK now surfaces HTTP 429 and retries login with a dedicated budget that honors `Retry-After`. API-key auth is unaffected (it skips login) (#231)

---

## [v0.44.0] - 2026-06-10

### ✨ Features

- **Site-level IGMP snooping** — manage the `igmp_snooping` site setting (`enabled` + `network_ids`) through the `unifi_setting` resource. On UniFi Network 10.3.x+ the effective IGMP snooping toggle moved from the per-network object to this site setting; advanced querier/flood options configured in the UI are preserved across updates (#164)

### 🐛 Bug Fixes

- `unifi_firewall_policy`: round-trip `connection_states` so a policy whose `connection_state_type` is `CUSTOM` updates successfully — the provider previously sent an empty state list and the firmware rejected it with HTTP 400 (#227)

---

## [v0.43.1] - 2026-06-10

### ✨ Features

- `unifi_radius_user`: derive the assigned VLAN from `network_id`, so MAC-based authentication (MAB) hands out the correct VLAN without hand-setting the tunnel attributes (#226)
- `unifi_radius_user`: support moving a deprecated `unifi_account` in place via a `moved` block — no more destroy/recreate or hand-edited state when migrating, since both are backed by the same RADIUS account (#222, #224)

### 🐛 Bug Fixes

- `unifi_firewall_policy`: round-trip the firmware-required fields (`connection_state_type`, `icmp_typename`, `icmp_v6_typename`, and the source/destination `matching_target_type`) so a zone-based UPDATE no longer fails with HTTP 400 on UniFi OS 5.1.x / Network 10.x (#220, #221, #223)
- `unifi_device`: write `op_mode` for non-default ports so SFP+ link aggregation (LAG) actually forms, while still skipping it on gateways (UDM) that reject `op_mode` on a PUT (#213, #225)

---

## [v0.43.0] - 2026-06-09

### ✨ Features

- **New `unifi_wireguard_peer` resource** — manage WireGuard VPN peers (the "clients" of a WireGuard server network), with full CRUD and import (#194)
- **New `unifi_firewall_zone` resource** — create and manage zone-based firewall zones (UniFi OS 8.x+) and their network membership, alongside the existing data source (#214, #218)
- **IPv6 network configuration** on `unifi_network` — static IPv6 subnet, Router Advertisement (`ipv6_ra*`), Prefix Delegation (`ipv6_pd_*`) and a DHCPv6 server block (#158)
- **WLAN private pre-shared keys (PPSK)** — per-key passphrases each optionally bound to a network/VLAN (#47, #212)
- **WLAN write-only passphrase** `passphrase_wo` (Terraform 1.11+) so the secret is used at apply time but never persisted to state (#201)

### 🐛 Bug Fixes

- `unifi_device`: read `radio_table` `channel`/`tx_power` returned as numbers by UniFi 10.x controllers — previously broke device read/import with an unmarshal error (#112)
- `unifi_device`: stop resetting `state`/`adopted` in the update payload, fixing writes on UDM / Dream Machine gateways (#177)
- `unifi_network`: keep `dhcp_relay` enabled by pinning a manual `setting_preference` (#208)
- `unifi_network`: stop forcing `multicast_dns = true` at create, which caused an "inconsistent result after apply" on UniFi OS gateways (#209)
- `unifi_network`: make `subnet` optional for vlan-only networks (#124)
- `unifi_network`: tolerate string-encoded boolean flags such as `dhcpd_enabled` from some controllers (#65)
- `unifi_network`: send `vlan_enabled` so create/update with a VLAN no longer fails with `api.err.VlanUsed` (#76, #85)
- `unifi_port_forward`: stop perpetual drift when the `source_limiting` block is omitted (#187)
- `unifi_firewall_policy`: support SPECIFIC port matching via a `port` attribute (#207)
- `unifi_wlan`: stop `mac_filter` drift, populate `wlangroup_id`, and stabilize `minimum_data_rate` (#200, #203)
- `unifi_dns_record`: make `record_type` required (#197)
- `unifi_port_profile`: expose forward/native/tagged VLANs in the data source schema (#196)
- `unifi_radius_user`: allow `tunnel_type` 13 (VLAN) (#193)
- `unifi_client`: zero-diff import/create for `blocked`/groups/`qos_rate` (#174)
- `unifi_client_info`: don't fail with 404 on controllers where the active-clients endpoint is unavailable (#121)
- structured logging via a dedicated subsystem (#168)

### 🔧 Build & CI

- run `gosec` on dependabot PRs and on `go.mod`/`go.sum` changes so dependency bumps can satisfy the code-scanning gate (#204, #205)
- dependency updates: testcontainers/compose, terraform-plugin-testing, grouped go modules, and GitHub Actions (#206, #166)

### 📄 Documentation

- clarify what `lte_lan` does (#202)
- document that `ipv6_pd_start`/`ipv6_pd_stop` are required for prefix-delegation networks (#215)

---

## [v0.41.20] - 2026-03-08

### 💥 Breaking Changes

#### `unifi_network` Resource Replaced

The `unifi_network` resource has been replaced with the modernized `unifi_virtual_network` implementation, which is now renamed to `unifi_network`.

**What changed:**

* The old `unifi_network` resource (flat attribute schema with `purpose`, `vlan_id`, `dhcp_start`, `dhcp_stop`, etc.) has been removed
* The `unifi_virtual_network` resource has been renamed to `unifi_network`
* The new `unifi_network` resource uses nested attributes (`dhcp_server`, `dhcp_relay`, `dhcp_guarding`) instead of flat prefixed fields
* The `unifi_network` data source is unchanged

**Migration guide:**

* Replace `purpose = "corporate"` — the new resource defaults to corporate purpose
* Replace `vlan_id` with `vlan`
* Replace `subnet` value format — now uses `cidrtypes.IPv4Prefix` (e.g., `"192.168.1.1/24"`)
* Replace flat DHCP fields (`dhcp_start`, `dhcp_stop`, `dhcp_enabled`) with nested `dhcp_server` block
* Replace `purpose = "vlan-only"` with `third_party_gateway = true`
* Remove `purpose`, `network_group`, and `vlan_enabled` attributes (no longer needed)
* WAN-specific attributes are no longer part of this resource — use `unifi_wan` instead

**Example migration:**

```hcl
# Before (old unifi_network)
resource "unifi_network" "vlan" {
  name         = "my-vlan"
  purpose      = "corporate"
  subnet       = "10.0.0.1/24"
  vlan_id      = 10
  dhcp_start   = "10.0.0.6"
  dhcp_stop    = "10.0.0.254"
  dhcp_enabled = true
}

# After (new unifi_network)
resource "unifi_network" "vlan" {
  name   = "my-vlan"
  subnet = "10.0.0.1/24"
  vlan   = 10

  dhcp_server = {
    enabled = true
    start   = "10.0.0.6"
    stop    = "10.0.0.254"
  }
}
```

**Other changes:**

* **Removed**: Old `unifi_network` resource and tests
* **Updated**: Examples for `unifi_client`, `unifi_port_profile`, and `unifi_wlan` to use new schema
* **Updated**: Documentation regenerated with new schema and examples

---

## [v0.41.19] - 2026-03-07

### 🔧 Improvements

#### Client Resource Enhancements

This release adds bulk import capability to the `unifi_client` resource, building on the expanded client list support introduced in v0.41.18.

**Changes**

* **New Example**: Added bulk import example (`examples/resources/unifi_client/bulk-import.tf`)
  * Demonstrates how to manage multiple client devices using a tfquery data file
* **New Example**: Added bulk import tfquery configuration (`examples/resources/unifi_client/bulk-import.tfquery.hcl`)
* **Improved**: Enhanced `unifi_client` resource with additional attributes and fixes
* **Docs**: Updated client list resource and port action documentation

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.18...v0.41.19>

---

## [v0.41.18] - 2026-03-07

### 🚀 New Features

#### New Data Sources

This release introduces two new list-style data sources for querying UniFi network clients and network member groups.

##### `unifi_client_list` (List Data Source)

A new list data source that provides a rich, queryable view of all UniFi network clients.

* Query and filter clients by various attributes
* Supports bulk operations and data-driven configurations
* Includes comprehensive tests

##### `unifi_network_members_group_list` (Data Source)

A new data source for listing network member groups.

**Other Changes**

* **Improved**: Enhanced `unifi_client` resource with additional attributes (158 additions)
* **Updated**: go-unifi dependency version bump
* **Fixed**: Minor fixes to `unifi_virtual_network_resource` and `unifi_vpn_client_resource`
* **Added**: New data source examples for `unifi_client_list` and `unifi_network_members_group_list`

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.17...v0.41.18>

---

## [v0.41.17] - 2026-02-26

### 🐛 Bug Fixes

#### Dynamic DNS Identity Field Fix

* **Fixed**: `bug: Fix identity in dynamic dns` — corrected the identity field in the Dynamic DNS resource that was broken since v0.41.13

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.16...v0.41.17>

---

## [v0.41.16] - 2026-02-26

### 🐛 Bug Fixes

#### UniFi Client Fix

* **Fixed**: Additional fixes to the `unifi_client` resource following the v0.41.15 update

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.15...v0.41.16>

---

## [v0.41.15] - 2026-02-26

### 🐛 Bug Fixes

#### UniFi Client Update

* **Fixed**: Updated `unifi_client` resource to resolve issues introduced in v0.41.13

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.14...v0.41.15>

---

## [v0.41.14] - 2026-02-26

### 🐛 Bug Fixes

#### Network Data Source Fix

* **Fixed**: `bug: Fix Network Data Source` — resolved a regression in the `unifi_network` data source introduced in v0.41.13

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.13...v0.41.14>

---

## [v0.41.13] - 2026-02-22

### 🔧 Maintenance

#### go-unifi Dependency Update and Provider Refactor

This release updates the go-unifi client library and significantly refactors the provider configuration code.

**Changes**

* **Updated**: go-unifi dependency version bump
* **Refactored**: Significant cleanup of `provider.go` (removed 92 lines of legacy code, -81 net lines)
* **Updated**: Provider tests updated to reflect new provider configuration
* **Fixed**: Minor fixes to `setting_resource.go`

> ⚠️ **Warning**: This release introduced regressions that were fixed in v0.41.14–v0.41.17:
>
> * **Network Data Source** had issues (fixed in v0.41.14)
> * **UniFi Client** had issues (fixed in v0.41.15–v0.41.16)
> * **Dynamic DNS** identity field was broken (fixed in v0.41.17)
>
> **Upgrade recommendation**: If upgrading from v0.41.12, skip directly to v0.41.17 or later.

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.12...v0.41.13>

---

## [v0.41.12] - 2026-01-25

### 🐛 Bug Fixes & 📄 Documentation

#### Client Data Source Fix and Documentation Update

* **Fixed**: `bug: Fix client data source` — resolved field mapping issues in the `unifi_client` data source
* **Fixed**: `Fix pointer` — corrected a nil pointer issue
* **Docs**: Updated generated documentation

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.11...v0.41.12>

---

## [v0.41.11] - 2026-01-25

### 🐛 Bug Fixes

#### DNS Port Fix

* **Fixed**: `bug: Fix DNS port` — corrected the port used for DNS queries in the provider

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.10...v0.41.11>

---

## [v0.41.10] - 2026-01-22

### 🐛 Bug Fixes

#### go-unifi Version Fix

* **Fixed**: `bug: Fix go-unifi version` — pinned the correct go-unifi dependency version to resolve compatibility issues introduced in v0.41.9

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.9...v0.41.10>

---

## [v0.41.9] - 2026-01-22

### 🚀 New Features & 🔧 Improvements

#### New WireGuard VPN Client Resource, WAN/WLAN Refactoring, and Expanded Tests

This release adds the `unifi_vpn_client` resource for WireGuard VPN configuration, refactors the WAN and WLAN resources for better code quality, and significantly expands test coverage.

**New Features**

* **NEW**: `unifi_vpn_client` resource (`unifi/vpn_client_resource.go`, 667 lines)
  * WireGuard VPN client configuration support
  * Dual configuration modes:
    * **File mode**: Upload a complete WireGuard configuration file
    * **Manual mode**: Configure peer settings directly (public key, endpoint, allowed IPs)
  * DNS servers support (1–2 entries required in manual mode)
  * Auto-mode detection based on nested configuration structure
  * Preshared key support for enhanced security
  * Sensitive field handling for private keys and configuration content
  * Flexible import formats: `id`, `name=<name>`, `site:id`, `site:name=<name>`
  * Complete CRUD operations with comprehensive error handling

**Improvements**

* **WAN Resource Refactoring**: Migrated to pointer-based API calls, simplified null value handling, reduced code verbosity (net -22 lines)
* **WLAN Resource Refactoring**: Converted to pointer-based API patterns, cleaner enabled state checks (net -16 lines)

**Testing**

* **New**: VPN client acceptance tests (file mode, manual mode with DNS, preshared key)
* **New**: Virtual network acceptance tests (basic VLAN, DHCP server, guest network)

**Files Changed**

* `unifi/vpn_client_resource.go` (NEW, 667 lines)
* `unifi/vpn_client_resource_test.go` (NEW, 211 lines)
* `unifi/virtual_network_resource_test.go` (NEW, 185 lines)
* `unifi/wan_resource.go` (+242/-264)
* `unifi/wlan_resource.go` (+11/-27)

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.8...v0.41.9>

---

## [v0.41.8] - 2026-01-16

### 🔧 Dependency Updates

#### Security and Dependency Bumps

* **Updated**: `github/codeql-action` from 3.29.0 to 4.31.10 (major version bump via Dependabot)
* **Updated**: `github.com/containerd/containerd/v2` from 2.1.4 to 2.1.5 (security patch, indirect dependency)

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.7...v0.41.8>

---

## [v0.41.7] - 2026-01-16

### 🔧 Improvements

#### CodeQL Security Scanning and Query/Actions Fixes

* **Added**: CodeQL analysis workflow configuration for automated security scanning
* **Fixed**: `feat: Fix query and actions` — resolved issues with list resource queries and action handling
* **Fixed**: `chore: Fix formatting and generation` — corrected code formatting and regenerated provider documentation

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.6...v0.41.7>

---

## [v0.41.6] - 2026-01-16

### 🚀 New Features

#### Client Info Data Source

* **Added**: `feat: Added Client Info` — new `unifi_client_info` data source for retrieving detailed information about a specific network client by MAC address or hostname

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.5...v0.41.6>

---

## [v0.41.5] - 2026-01-15

### 🐛 Bug Fixes & Build Improvements

#### Client Info Data Source Fix and GoReleaser Update

This release fixes the `unifi_client_info` data source and updates the release pipeline for proper Terraform Registry integration.

**Changes**

* **Fixed**: `unifi_client_info` data source field mapping and model alignment
* **Updated**: GoReleaser configuration with Terraform Registry support
* **Added**: `terraform-registry-manifest.json` for proper Terraform Registry integration
  * This enables correct discovery by the Terraform Registry

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4...v0.41.5>

---

## [v0.41.4] - 2026-01-15

### 🚀 New Features

#### Terraform Plugin Framework Migration (Stable Release) and Client Info Data Sources

This is the stable release of the Terraform Plugin Framework migration, incorporating all the work from the beta and RC pre-releases.

**Changes since v0.41.3**

* **Migrated**: Full provider migration from Terraform Plugin SDK v2 to Terraform Plugin Framework via the MUX adapter — allows both old SDK resources and new Framework resources to coexist
* **Added**: `unifi_client_info` data source (single-client lookup by MAC/hostname)
* **Added**: `unifi_client_info_list` data source (bulk client info queries)
* **Breaking**: `unifi_user` renamed to `unifi_client`; `unifi_user_group` renamed to `unifi_client_group`
* **Added**: `unifi_wan` resource for full WAN interface configuration
* **Improved**: `unifi_wlan` resource with major schema and behavior improvements
* **Added**: Structured logging via `unifi/logger.go`
* **Fixed**: GoReleaser configuration and Terraform Registry manifest

## What's Changed

* Pivot to Plugin Framework via the MUX Framework by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17>
* feat: Migrate to Terraform plugin framework by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/50>

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4>

---

## [v0.41.4-rc3] - 2026-01-06

### ⚠️ BREAKING CHANGES

#### Renamed `unifi_user` → `unifi_client` and `unifi_user_group` → `unifi_client_group`

This release candidate introduces a **breaking rename** of the user-related resources and data sources to better reflect their purpose in UniFi terminology.

**Breaking Changes**

| Old Name | New Name |
|----------|----------|
| `unifi_user` (resource) | `unifi_client` (resource) |
| `unifi_user_group` (resource) | `unifi_client_group` (resource) |
| `unifi_user` (data source) | `unifi_client` (data source) |
| `unifi_user_group` (data source) | `unifi_client_group` (data source) |

> **Migration**: Replace all references to `unifi_user` with `unifi_client` and `unifi_user_group` with `unifi_client_group` in your Terraform configurations.

**Other Changes**

* **Added**: WAN resource (`unifi_wan`) documentation and import examples
* **Updated**: go-unifi dependency

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-rc2...v0.41.4-rc3>

---

## [v0.41.4-rc2] - 2026-01-06

### 🚀 New Features & Bug Fixes

#### New WAN Resource, WLAN Improvements, and Acceptance Test Fixes

This release candidate adds the `unifi_wan` resource, significantly improves `unifi_wlan`, and fixes the acceptance test suite for the new plugin framework.

**New Features**

* **NEW**: `unifi_wan` resource (`unifi/wan_resource.go`, ~1129 lines)
  * Full WAN interface configuration management
  * Import support
  * Comprehensive documentation
* **Improved**: `unifi_wlan` resource with major enhancements (319 additions)
* **Added**: Structured logging (`unifi/logger.go`)
* **Improved**: `unifi_network` resource with bug fixes and schema improvements

**Bug Fixes**

* Fixed acceptance tests to work with the new plugin framework
* Updated `unifi_site` resource with framework compatibility fixes
* Updated Dependabot configuration for automated dependency management

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-rc1...v0.41.4-rc2>

---

## [v0.41.4-rc1] - 2025-12-31

### 🚀 Release Candidate: Terraform Plugin Framework Migration

This release candidate marks the first RC of the full migration from Terraform Plugin SDK v2 to the Terraform Plugin Framework, delivered via the MUX adapter so old and new resource implementations can coexist.

**Changes**

* **Migrated**: Provider core pivoted to Terraform Plugin Framework via the MUX (protocol multiplexer) adapter
* **Maintained**: Full backward compatibility with all existing resources during the migration period

## What's Changed

* Pivot to Plugin Framework via the MUX Framework by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17>
* feat: Migrate to Terraform plugin framework by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/50>

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4-rc1>

---

## [v0.41.4-beta2] - 2025-11-18

### 🔧 Improvements

#### Optional Provider Configuration

This beta release makes provider configuration fields optional, allowing more flexible authentication configuration via environment variables.

**Changes**

* **Improved**: Provider configuration fields are now optional (previously required)
  * All fields can now be configured via environment variables (`UNIFI_API_URL`, `UNIFI_USERNAME`, `UNIFI_PASSWORD`, `UNIFI_API_KEY`, etc.)
  * This enables cleaner CI/CD configurations without hardcoded provider blocks
* **Added**: Expanded `unifi_device` resource documentation with full attribute reference

> **Note**: This is a beta release for the Terraform Plugin Framework migration. See v0.41.4-beta1 for the full feature list.

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-beta1...v0.41.4-beta2>

---

## [v0.41.4-beta1] - 2025-11-18

### 🧪 Beta: Terraform Plugin Framework Migration

Initial beta release of the Terraform Plugin Framework migration. This beta introduces the new plugin framework architecture while maintaining compatibility with all existing resources.

**Changes**

* **Migrated**: Provider core to Terraform Plugin Framework via the MUX adapter
* **Refactored**: Multiple resources updated to use the new plugin framework patterns

## What's Changed

* Pivot to Plugin Framework via the MUX Framework by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17>
* Plugin-framework-migration by @appkins in <https://github.com/ubiquiti-community/terraform-provider-unifi/pull/19>

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4-beta1>

---

## [v0.41.3] - 2025-06-19

### 🚀 New Features

#### API Key Authentication Support and Code Quality Improvements

This release introduces **API Key authentication** as an alternative to username/password authentication, providing enhanced security and convenience for automated deployments. It also includes extensive code quality improvements across the provider.

**New Features**

* **New `api_key` provider configuration**: Authenticate using API keys instead of username/password
* **Environment variable support**: Use `UNIFI_API_KEY` environment variable for CI/CD pipelines
* **Automatic fallback**: When API key is provided, username/password fields are ignored
* **Validation**: API keys are validated to ensure they meet minimum length requirements (32+ characters)

```terraform
provider "unifi" {
  api_key = var.api_key    # or use UNIFI_API_KEY env var
  api_url = var.api_url
  site    = "default"
}
```

**Code Quality Improvements**

* **Fixed 60+ golangci-lint issues** across data sources and resources
* **Enhanced type safety**: All type assertions now include proper error checking to prevent runtime panics
* **Improved error handling**: Return values from `d.Set()` calls are now properly handled
* **Parameter validation**: Function parameters validated with appropriate error messages

**Migration from Username/Password**

Existing configurations using username/password will continue to work unchanged. This release is **fully backward compatible**.

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.2...v0.41.3>

---

## [v0.41.2] - 2024-07-31

### 🔧 Build & Release Fixes

#### GoReleaser and Workflow Updates

* **Updated**: GoReleaser configuration to fix release artifact generation
* **Updated**: Release workflow permissions and configuration
* **Fixed**: Version bump and cleanup of release tooling

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.1...v0.41.2>

---

## [v0.41.1] - 2024-07-31

### 🚀 Initial Release of Community Fork

#### DNS Record Resource, WireGuard, and Provider Modernization

This is the initial release of the `ubiquiti-community/terraform-provider-unifi` fork, establishing the project foundation with new resources, updated tooling, and a clean dependency structure.

**New Features**

* **Added**: `unifi_dns_record` resource for managing DNS records in UniFi controllers
* **Added**: WireGuard VPN configuration support
* **Updated**: DNS record resource with improved field handling

**Infrastructure**

* **Updated**: Go module versions and dependency versions
* **Removed**: Vendored dependencies in favor of Go modules
* **Added**: Dependabot configuration for automated dependency management
* **Updated**: Release workflow permissions
* **Added**: Provider documentation

**Full Changelog**: <https://github.com/ubiquiti-community/terraform-provider-unifi/commits/v0.41.1>
