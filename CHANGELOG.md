# Changelog

All notable changes to this project will be documented in this file.

## [v0.41.19] - 2026-03-07

### 🔧 Improvements

#### Client Resource Enhancements

This release adds bulk import capability to the `unifi_client` resource, building on the expanded client list support introduced in v0.41.18.

**Changes**

* **New Example**: Added bulk import example (`examples/resources/unifi_client/bulk-import.tf`)
  - Demonstrates how to manage multiple client devices using a tfquery data file
* **New Example**: Added bulk import tfquery configuration (`examples/resources/unifi_client/bulk-import.tfquery.hcl`)
* **Improved**: Enhanced `unifi_client` resource with additional attributes and fixes
* **Docs**: Updated client list resource and port action documentation

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.18...v0.41.19

---

## [v0.41.18] - 2026-03-07

### 🚀 New Features

#### New Data Sources

This release introduces two new list-style data sources for querying UniFi network clients and network member groups.

##### `unifi_client_list` (List Data Source)

A new list data source that provides a rich, queryable view of all UniFi network clients.

- Query and filter clients by various attributes
- Supports bulk operations and data-driven configurations
- Includes comprehensive tests

##### `unifi_network_members_group_list` (Data Source)

A new data source for listing network member groups.

**Other Changes**

* **Improved**: Enhanced `unifi_client` resource with additional attributes (158 additions)
* **Updated**: go-unifi dependency version bump
* **Fixed**: Minor fixes to `unifi_virtual_network_resource` and `unifi_vpn_client_resource`
* **Added**: New data source examples for `unifi_client_list` and `unifi_network_members_group_list`

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.17...v0.41.18

---

## [v0.41.17] - 2026-02-26

### 🐛 Bug Fixes

#### Dynamic DNS Identity Field Fix

* **Fixed**: `bug: Fix identity in dynamic dns` — corrected the identity field in the Dynamic DNS resource that was broken since v0.41.13

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.16...v0.41.17

---

## [v0.41.16] - 2026-02-26

### 🐛 Bug Fixes

#### UniFi Client Fix

* **Fixed**: Additional fixes to the `unifi_client` resource following the v0.41.15 update

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.15...v0.41.16

---

## [v0.41.15] - 2026-02-26

### 🐛 Bug Fixes

#### UniFi Client Update

* **Fixed**: Updated `unifi_client` resource to resolve issues introduced in v0.41.13

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.14...v0.41.15

---

## [v0.41.14] - 2026-02-26

### 🐛 Bug Fixes

#### Network Data Source Fix

* **Fixed**: `bug: Fix Network Data Source` — resolved a regression in the `unifi_network` data source introduced in v0.41.13

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.13...v0.41.14

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
> - **Network Data Source** had issues (fixed in v0.41.14)
> - **UniFi Client** had issues (fixed in v0.41.15–v0.41.16)
> - **Dynamic DNS** identity field was broken (fixed in v0.41.17)
>
> **Upgrade recommendation**: If upgrading from v0.41.12, skip directly to v0.41.17 or later.

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.12...v0.41.13

---

## [v0.41.12] - 2026-01-25

### 🐛 Bug Fixes & 📄 Documentation

#### Client Data Source Fix and Documentation Update

* **Fixed**: `bug: Fix client data source` — resolved field mapping issues in the `unifi_client` data source
* **Fixed**: `Fix pointer` — corrected a nil pointer issue
* **Docs**: Updated generated documentation

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.11...v0.41.12

---

## [v0.41.11] - 2026-01-25

### 🐛 Bug Fixes

#### DNS Port Fix

* **Fixed**: `bug: Fix DNS port` — corrected the port used for DNS queries in the provider

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.10...v0.41.11

---

## [v0.41.10] - 2026-01-22

### 🐛 Bug Fixes

#### go-unifi Version Fix

* **Fixed**: `bug: Fix go-unifi version` — pinned the correct go-unifi dependency version to resolve compatibility issues introduced in v0.41.9

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.9...v0.41.10

---

## [v0.41.9] - 2026-01-22

### 🚀 New Features & 🔧 Improvements

#### New WireGuard VPN Client Resource, WAN/WLAN Refactoring, and Expanded Tests

This release adds the `unifi_vpn_client` resource for WireGuard VPN configuration, refactors the WAN and WLAN resources for better code quality, and significantly expands test coverage.

**New Features**

* **NEW**: `unifi_vpn_client` resource (`unifi/vpn_client_resource.go`, 667 lines)
  - WireGuard VPN client configuration support
  - Dual configuration modes:
    * **File mode**: Upload a complete WireGuard configuration file
    * **Manual mode**: Configure peer settings directly (public key, endpoint, allowed IPs)
  - DNS servers support (1–2 entries required in manual mode)
  - Auto-mode detection based on nested configuration structure
  - Preshared key support for enhanced security
  - Sensitive field handling for private keys and configuration content
  - Flexible import formats: `id`, `name=<name>`, `site:id`, `site:name=<name>`
  - Complete CRUD operations with comprehensive error handling

**Improvements**

* **WAN Resource Refactoring**: Migrated to pointer-based API calls, simplified null value handling, reduced code verbosity (net -22 lines)
* **WLAN Resource Refactoring**: Converted to pointer-based API patterns, cleaner enabled state checks (net -16 lines)

**Testing**

* **New**: VPN client acceptance tests (file mode, manual mode with DNS, preshared key)
* **New**: Virtual network acceptance tests (basic VLAN, DHCP server, guest network)

**Files Changed**

- `unifi/vpn_client_resource.go` (NEW, 667 lines)
- `unifi/vpn_client_resource_test.go` (NEW, 211 lines)
- `unifi/virtual_network_resource_test.go` (NEW, 185 lines)
- `unifi/wan_resource.go` (+242/-264)
- `unifi/wlan_resource.go` (+11/-27)

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.8...v0.41.9

---

## [v0.41.8] - 2026-01-16

### 🔧 Dependency Updates

#### Security and Dependency Bumps

* **Updated**: `github/codeql-action` from 3.29.0 to 4.31.10 (major version bump via Dependabot)
* **Updated**: `github.com/containerd/containerd/v2` from 2.1.4 to 2.1.5 (security patch, indirect dependency)

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.7...v0.41.8

---

## [v0.41.7] - 2026-01-16

### 🔧 Improvements

#### CodeQL Security Scanning and Query/Actions Fixes

* **Added**: CodeQL analysis workflow configuration for automated security scanning
* **Fixed**: `feat: Fix query and actions` — resolved issues with list resource queries and action handling
* **Fixed**: `chore: Fix formatting and generation` — corrected code formatting and regenerated provider documentation

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.6...v0.41.7

---

## [v0.41.6] - 2026-01-16

### 🚀 New Features

#### Client Info Data Source

* **Added**: `feat: Added Client Info` — new `unifi_client_info` data source for retrieving detailed information about a specific network client by MAC address or hostname

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.5...v0.41.6

---

## [v0.41.5] - 2026-01-15

### 🐛 Bug Fixes & Build Improvements

#### Client Info Data Source Fix and GoReleaser Update

This release fixes the `unifi_client_info` data source and updates the release pipeline for proper Terraform Registry integration.

**Changes**

* **Fixed**: `unifi_client_info` data source field mapping and model alignment
* **Updated**: GoReleaser configuration with Terraform Registry support
* **Added**: `terraform-registry-manifest.json` for proper Terraform Registry integration
  - This enables correct discovery by the Terraform Registry

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4...v0.41.5

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
* Pivot to Plugin Framework via the MUX Framework by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17
* feat: Migrate to Terraform plugin framework by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/50

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4

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

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-rc2...v0.41.4-rc3

---

## [v0.41.4-rc2] - 2026-01-06

### 🚀 New Features & Bug Fixes

#### New WAN Resource, WLAN Improvements, and Acceptance Test Fixes

This release candidate adds the `unifi_wan` resource, significantly improves `unifi_wlan`, and fixes the acceptance test suite for the new plugin framework.

**New Features**

* **NEW**: `unifi_wan` resource (`unifi/wan_resource.go`, ~1129 lines)
  - Full WAN interface configuration management
  - Import support
  - Comprehensive documentation
* **Improved**: `unifi_wlan` resource with major enhancements (319 additions)
* **Added**: Structured logging (`unifi/logger.go`)
* **Improved**: `unifi_network` resource with bug fixes and schema improvements

**Bug Fixes**

* Fixed acceptance tests to work with the new plugin framework
* Updated `unifi_site` resource with framework compatibility fixes
* Updated Dependabot configuration for automated dependency management

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-rc1...v0.41.4-rc2

---

## [v0.41.4-rc1] - 2025-12-31

### 🚀 Release Candidate: Terraform Plugin Framework Migration

This release candidate marks the first RC of the full migration from Terraform Plugin SDK v2 to the Terraform Plugin Framework, delivered via the MUX adapter so old and new resource implementations can coexist.

**Changes**

* **Migrated**: Provider core pivoted to Terraform Plugin Framework via the MUX (protocol multiplexer) adapter
* **Maintained**: Full backward compatibility with all existing resources during the migration period

## What's Changed
* Pivot to Plugin Framework via the MUX Framework by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17
* feat: Migrate to Terraform plugin framework by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/50

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4-rc1

---

## [v0.41.4-beta2] - 2025-11-18

### 🔧 Improvements

#### Optional Provider Configuration

This beta release makes provider configuration fields optional, allowing more flexible authentication configuration via environment variables.

**Changes**

* **Improved**: Provider configuration fields are now optional (previously required)
  - All fields can now be configured via environment variables (`UNIFI_API_URL`, `UNIFI_USERNAME`, `UNIFI_PASSWORD`, `UNIFI_API_KEY`, etc.)
  - This enables cleaner CI/CD configurations without hardcoded provider blocks
* **Added**: Expanded `unifi_device` resource documentation with full attribute reference

> **Note**: This is a beta release for the Terraform Plugin Framework migration. See v0.41.4-beta1 for the full feature list.

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.4-beta1...v0.41.4-beta2

---

## [v0.41.4-beta1] - 2025-11-18

### 🧪 Beta: Terraform Plugin Framework Migration

Initial beta release of the Terraform Plugin Framework migration. This beta introduces the new plugin framework architecture while maintaining compatibility with all existing resources.

**Changes**

* **Migrated**: Provider core to Terraform Plugin Framework via the MUX adapter
* **Refactored**: Multiple resources updated to use the new plugin framework patterns

## What's Changed
* Pivot to Plugin Framework via the MUX Framework by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/17
* Plugin-framework-migration by @appkins in https://github.com/ubiquiti-community/terraform-provider-unifi/pull/19

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.3...v0.41.4-beta1

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

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.2...v0.41.3

---

## [v0.41.2] - 2024-07-31

### 🔧 Build & Release Fixes

#### GoReleaser and Workflow Updates

* **Updated**: GoReleaser configuration to fix release artifact generation
* **Updated**: Release workflow permissions and configuration
* **Fixed**: Version bump and cleanup of release tooling

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/compare/v0.41.1...v0.41.2

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

**Full Changelog**: https://github.com/ubiquiti-community/terraform-provider-unifi/commits/v0.41.1
