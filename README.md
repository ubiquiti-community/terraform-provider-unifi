# Unifi Terraform Provider (terraform-provider-unifi)

[![Acceptance Tests](https://github.com/ubiquiti-community/terraform-provider-unifi/actions/workflows/acctest.yaml/badge.svg)](https://github.com/ubiquiti-community/terraform-provider-unifi/actions/workflows/acctest.yaml) [![codecov](https://codecov.io/github/ubiquiti-community/terraform-provider-unifi/graph/badge.svg?token=KVP7FS41IG)](https://codecov.io/github/ubiquiti-community/terraform-provider-unifi)

> **Note**: You can't (for obvious reasons) configure your network while connected to something that may disconnect (like the WiFi). Use a hard-wired connection to your controller to use this provider.

Functionality first needs to be added to the [go-unifi](https://github.com/ubiquiti-community/go-unifi) SDK.

## Documentation

You can browse documentation on the [Terraform provider registry](https://registry.terraform.io/providers/paultyng/unifi/latest/docs).

## Supported Unifi Controller Versions

As of version [v0.34](https://github.com/ubiquiti-community/terraform-provider-unifi/releases/tag/v0.34.0), this provider only supports version 6 of the Unifi controller software. If you need v5 support, you can pin an older version of the provider.

The docker, UDM, and UDM-Pro versions are slightly different (the API is proxied a little differently) but for the most part should all be supported. Individual patch versions of the controller are generally not tested for compatibility, just the latest stable versions.

## Using the Provider

### Terraform 1.0 and above

You can use the provider via the [Terraform provider registry](https://registry.terraform.io/providers/paultyng/unifi).

## Acceptance Tests

`TF_ACC=1 go test ./unifi/...` boots the demo-mode controller from `docker-compose.yaml` via testcontainers; a Docker (or Podman) socket is the only prerequisite. The compose file also starts a `unifi-device-sim` sidecar — emulated UniFi devices speaking the real inform protocol, from [unifi-emu](https://github.com/jamesbraid/unifi-emu) — because controllers without demo mode (for example a seeded UniFi OS appliance) expose no devices for the device tests to adopt. The image is a local build until one is published: `docker build -t unifi-emu:dev .` in the unifi-emu repo. With the default demo-mode controller the sidecar's devices are extra inventory the tests don't assert on, so the sidecar just runs; its default MACs deliberately avoid `00:27:22:00:00:02`, which the demo seeder already presents. If the harness ever swaps to a device-less seeded controller, declare that MAC in `SIM_DEVICES` so the device test's adoption contract keeps holding. `UNIFI_SKIP_CONTAINER=1` skips compose entirely and tests against `UNIFI_API` — in that mode, run your own sim against the external controller or skip the device tests.
