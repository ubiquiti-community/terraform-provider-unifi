# Handoff: unifi_network import/refresh bugs (blocks ubitofu E2E suite)

**For:** a session fixing `unifi_network`'s import path on this fork —
prime upstream-PR material, same class as the `unifi_device`
carry-dropped-fields fix (a7bcfc4f).

**Found by:** ubitofu's live E2E gate (branch `emdash/testing-6agni`),
2026-07-22, against registry **v0.55.0** and a classic UniFi Network
**10.4.57** controller (`ghcr.io/jamesbraid/unifi-network:10.4.57-seeded`).
Deterministic, 3/3 fresh-site runs. ubitofu's write scenarios are parked
on these; its `docs/provider-import-bugs.md` carries the consumer-side
record and un-parking checklist. On fix, ubitofu S1 becomes a live
regression test.

## Bug 1 — import-refresh Read drops real attributes → spurious update

`ubitofu generate` emits HCL that byte-for-byte mirrors what the classic
REST API returns, plus `import {}` blocks. `tofu plan` then shows
`~ update in-place (imported from ...)` on **every** network, because the
provider's Read during import-refresh returns null/unset for attributes
the controller genuinely has:

```
  # unifi_network.default will be updated in-place
  # (imported from "...")
  ~ resource "unifi_network" "default" {
      ~ dhcp_server                    = {
          + leasetime           = "24h0m0s"
            ...
        }
      + gateway_type                   = "default"
      + setting_preference             = "auto"
    }

  # unifi_network.s1_net will be updated in-place
  ~ resource "unifi_network" "s1_net" {
      + gateway_type                  = "default"
      + ipv6_interface_type           = "none"
    }
```

Dropped set observed: `gateway_type`, `setting_preference`,
`dhcp_server.leasetime`, `ipv6_interface_type`. Live values confirmed
server-side (the rejected PUT payload itself carries
`"dhcpd_leasetime":86400`).

Where to look: the network resource's Read/flatten path vs the 10.x
wire fields. go.mod pins go-unifi `v1.33.43-0.20260706191309-bc63776a9ebf`;
go-unifi's `controller-testing` branch regenerated structs for 10.4.57
(`b17d01c unifi: regenerate for UniFi Network 10.4.57`,
`af942d5 unifi: restore dropped device fields`) — the same
stale-schema-vs-10.x pattern is the likely cause here.

## Bug 2 — consequence: a disabled default network is un-adoptable

`rest/networkconf` is a full-object PUT. The forced "no-op" update from
Bug 1 echoes the default network's real `enabled: false`, and the classic
controller rejects ANY default-network PUT carrying `enabled:false` —
changing or not:

```
Error Updating network
  with unifi_network.default,
api.err.DisablingDefaultNetworkNotAllowed (400) for PUT
  .../rest/networkconf/<id>
payload: {"_id": "...", ..., "enabled": false, ..., "name": "Default", ...}
```

Fixing Bug 1 removes the trigger; independently consider not sending
`enabled` for the default network when it is not changing (defense in
depth — same shape as the device dropped-fields fix).

## Bug 3 — domain_name null → "" consistency crash on ordinary networks

On the forced update of a freshly imported network whose config leaves
`domain_name` unset, the PUT response returns `domain_name: ""` and the
SDK's plan/apply consistency check aborts:

```
Error: Provider produced inconsistent result after apply
When applying changes to unifi_network.s1_net, provider ... produced an
unexpected new value: .domain_name: was null, but now cty.StringVal("").
This is a bug in the provider, ...
```

Fix direction: normalize `""` ↔ null for `domain_name` (plan modifier /
state normalization), as done for other Optional+Computed strings.

## Reproduction

Fastest (full harness): in ubitofu's branch, remove the `@pytest.mark.skip`
on `tests/controllertest/test_scenarios_reconcile.py::test_s1_in_sync_reconcile_exits_zero`
and run `pytest -m controller -k s1` (needs docker/colima; boots the
seeded image, seeds a site + one VLAN network, generates, applies).

Standalone (no ubitofu): run
`ghcr.io/jamesbraid/unifi-network:10.4.57-seeded` (admin /
`unifi-containers-seeded`, cookie login `POST /api/login`), create a site
(`cmd/sitemgr add-site`) and one network
(`POST /api/s/<site>/rest/networkconf` with
`{"name":"x","purpose":"corporate","vlan_enabled":true,"vlan":210,
"ip_subnet":"10.99.210.1/24","dhcpd_enabled":false}`), then a main.tf
with the provider (username/password/`allow_insecure`), config mirroring
the two networks' live values, and `import {}` blocks by `_id`.
`tofu plan` reproduces Bug 1's spurious updates; `tofu apply` reproduces
Bugs 2 and 3.

## Acceptance

- `tofu plan` right after import of an untouched network: **zero
  changes**.
- ubitofu S1 green (their `docs/provider-import-bugs.md` un-parking
  checklist), which then unlocks their S2–S10 live scenarios — a free
  ongoing regression suite for this fork.
