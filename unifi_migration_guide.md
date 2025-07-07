# Terraform Provider UniFi: SDK v2 to Plugin Framework Migration Guide

## Overview

This guide provides detailed instructions for migrating the terraform-provider-unifi from SDK v2 to the newer Plugin Framework. The provider is currently using a mixed approach with both SDK v2 and Plugin Framework providers running side-by-side using terraform-plugin-mux.

## Current State

The terraform-provider-unifi has already been set up with:

- **Mixed Provider Architecture**: Uses `tf6muxserver` to run both SDK v2 and Plugin Framework providers
- **Framework Foundation**: Basic Plugin Framework provider at `internal/provider/provider_framework.go`
- **Main Entry Point**: Updated `main.go` with mux server configuration
- **Dependencies**: Plugin Framework dependencies already included in `go.mod`

## Project Structure

### Current Structure
```
terraform-provider-unifi/
├── main.go                           # Mux server setup (SDK v2 + Framework)
├── internal/provider/
│   ├── provider.go                   # SDK v2 provider (existing resources)
│   ├── provider_framework.go         # Plugin Framework provider (empty)
│   ├── resource_network.go           # SDK v2 network resource
│   ├── resource_wlan.go              # SDK v2 WLAN resource
│   ├── resource_user.go              # SDK v2 user resource
│   └── [other SDK v2 resources]
├── docs/                             # Auto-generated documentation
└── examples/                         # Example configurations
```

### Target Structure (After Migration)
```
terraform-provider-unifi/
├── main.go                           # Framework-only provider
├── internal/provider/
│   ├── provider_framework.go         # Plugin Framework provider (all resources)
│   ├── resource_network_framework.go # Framework network resource
│   ├── resource_wlan_framework.go    # Framework WLAN resource
│   ├── resource_user_framework.go    # Framework user resource
│   └── util/
│       └── conversion.go             # Type conversion utilities
```

## Migration Strategy

### Phase 1: Infrastructure Setup ✅ (Complete)

The basic infrastructure is already in place:

1. **Dependencies**: Plugin Framework packages added to `go.mod`
2. **Mux Server**: Configured in `main.go` to serve both providers
3. **Framework Provider**: Basic structure in `provider_framework.go`

**Note**: You'll need to add the validators package when migrating resources with validation:
```bash
go get github.com/hashicorp/terraform-plugin-framework-validators
```

### Phase 2: Resource Migration (In Progress)

Priority order for migrating resources:

1. **unifi_network** - Core network management (most critical)
2. **unifi_wlan** - WiFi network configuration  
3. **unifi_user** - User management
4. **unifi_port_profile** - Port profile configuration
5. **unifi_firewall_rule** - Firewall rules
6. **unifi_site** - Site management

### Phase 3: Data Source Migration

Migrate corresponding data sources for each resource.

### Phase 4: Final Cleanup

Remove SDK v2 provider and mux setup once all resources are migrated.

## Migration Patterns

### 1. Resource Structure Template

For each SDK v2 resource (e.g., `resource_network.go`), create a Framework equivalent:

```go
// internal/provider/resource_network_framework.go
package provider

import (
    "context"
    "strings"
    
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    "github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-framework/path"
    
    "github.com/ubiquiti-community/go-unifi/unifi"
)

type networkFrameworkResource struct {
    client *client
}

type networkResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Site        types.String `tfsdk:"site"`
    Name        types.String `tfsdk:"name"`
    Purpose     types.String `tfsdk:"purpose"`
    Subnet      types.String `tfsdk:"subnet"`
    VLANId      types.Int64  `tfsdk:"vlan_id"`
    DHCPEnabled types.Bool   `tfsdk:"dhcp_enabled"`
    DHCPStart   types.String `tfsdk:"dhcp_start"`
    DHCPStop    types.String `tfsdk:"dhcp_stop"`
    // Add other fields from SDK v2 version
}

var (
    _ resource.Resource                = &networkFrameworkResource{}
    _ resource.ResourceWithConfigure   = &networkFrameworkResource{}
    _ resource.ResourceWithImportState = &networkFrameworkResource{}
)

func NewNetworkFrameworkResource() resource.Resource {
    return &networkFrameworkResource{}
}

func (r *networkFrameworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_network"
}

func (r *networkFrameworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        MarkdownDescription: "`unifi_network` manages WAN/LAN/VLAN networks.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                MarkdownDescription: "The ID of the network.",
                Computed:            true,
            },
            "site": schema.StringAttribute{
                MarkdownDescription: "The name of the site to associate the network with.",
                Optional:            true,
                Computed:            true,
            },
            "name": schema.StringAttribute{
                MarkdownDescription: "The name of the network.",
                Required:            true,
            },
            "purpose": schema.StringAttribute{
                MarkdownDescription: "The purpose of the network. Must be one of `corporate`, `guest`, `wan`, `vlan-only`, or `vpn-client`.",
                Required:            true,
                Validators: []validator.String{
                    stringvalidator.OneOf("corporate", "guest", "wan", "vlan-only", "vpn-client"),
                },
            },
            "subnet": schema.StringAttribute{
                MarkdownDescription: "The subnet of the network.",
                Optional:            true,
            },
            "vlan_id": schema.Int64Attribute{
                MarkdownDescription: "The VLAN ID of the network.",
                Optional:            true,
                Validators: []validator.Int64{
                    int64validator.Between(1, 4094),
                },
            },
            "dhcp_enabled": schema.BoolAttribute{
                MarkdownDescription: "Specifies whether DHCP is enabled.",
                Optional:            true,
                Computed:            true,
            },
            "dhcp_start": schema.StringAttribute{
                MarkdownDescription: "The IPv4 address where the DHCP range of addresses starts.",
                Optional:            true,
            },
            "dhcp_stop": schema.StringAttribute{
                MarkdownDescription: "The IPv4 address where the DHCP range of addresses stops.",
                Optional:            true,
            },
        },
    }
}

func (r *networkFrameworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    client, ok := req.ProviderData.(*client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            "Expected *client, got something else",
        )
        return
    }

    r.client = client
}

func (r *networkFrameworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan networkResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    site := plan.Site.ValueString()
    if site == "" {
        site = r.client.site
    }

    // Convert Framework model to API request
    createReq := &unifi.Network{
        Name:    plan.Name.ValueString(),
        Purpose: plan.Purpose.ValueString(),
        SiteID:  site,
    }

    if !plan.Subnet.IsNull() {
        createReq.Subnet = plan.Subnet.ValueString()
    }
    if !plan.VLANId.IsNull() {
        createReq.VLANId = int(plan.VLANId.ValueInt64())
    }
    if !plan.DHCPEnabled.IsNull() {
        createReq.DHCPEnabled = plan.DHCPEnabled.ValueBool()
    }
    if !plan.DHCPStart.IsNull() {
        createReq.DHCPStart = plan.DHCPStart.ValueString()
    }
    if !plan.DHCPStop.IsNull() {
        createReq.DHCPStop = plan.DHCPStop.ValueString()
    }

    // Create via API
    network, err := r.client.c.CreateNetwork(ctx, site, createReq)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error creating network",
            "Could not create network: "+err.Error(),
        )
        return
    }

    // Update plan with computed values
    plan.ID = types.StringValue(network.ID)
    plan.Site = types.StringValue(site)

    // Set state
    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}

func (r *networkFrameworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state networkResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    site := state.Site.ValueString()
    if site == "" {
        site = r.client.site
    }

    // Read from API
    network, err := r.client.c.GetNetwork(ctx, site, state.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError(
            "Error reading network",
            "Could not read network: "+err.Error(),
        )
        return
    }

    // Update state with API response
    r.updateModelFromAPI(ctx, &state, network, site)

    diags = resp.State.Set(ctx, state)
    resp.Diagnostics.Append(diags...)
}

// Critical: Handle null vs empty consistently to avoid import issues
func (r *networkFrameworkResource) updateModelFromAPI(ctx context.Context, model *networkResourceModel, network *unifi.Network, site string) {
    model.ID = types.StringValue(network.ID)
    model.Site = types.StringValue(site)
    model.Name = types.StringValue(network.Name)
    model.Purpose = types.StringValue(network.Purpose)

    // Handle optional fields - use null for empty values
    if network.Subnet == "" {
        model.Subnet = types.StringNull()
    } else {
        model.Subnet = types.StringValue(network.Subnet)
    }

    if network.VLANId == 0 {
        model.VLANId = types.Int64Null()
    } else {
        model.VLANId = types.Int64Value(int64(network.VLANId))
    }

    model.DHCPEnabled = types.BoolValue(network.DHCPEnabled)

    if network.DHCPStart == "" {
        model.DHCPStart = types.StringNull()
    } else {
        model.DHCPStart = types.StringValue(network.DHCPStart)
    }

    if network.DHCPStop == "" {
        model.DHCPStop = types.StringNull()
    } else {
        model.DHCPStop = types.StringValue(network.DHCPStop)
    }
}

func (r *networkFrameworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var plan networkResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    site := plan.Site.ValueString()
    if site == "" {
        site = r.client.site
    }

    // Convert Framework model to API request
    updateReq := &unifi.Network{
        ID:      plan.ID.ValueString(),
        Name:    plan.Name.ValueString(),
        Purpose: plan.Purpose.ValueString(),
        SiteID:  site,
    }

    // Set optional fields
    if !plan.Subnet.IsNull() {
        updateReq.Subnet = plan.Subnet.ValueString()
    }
    if !plan.VLANId.IsNull() {
        updateReq.VLANId = int(plan.VLANId.ValueInt64())
    }
    if !plan.DHCPEnabled.IsNull() {
        updateReq.DHCPEnabled = plan.DHCPEnabled.ValueBool()
    }
    if !plan.DHCPStart.IsNull() {
        updateReq.DHCPStart = plan.DHCPStart.ValueString()
    }
    if !plan.DHCPStop.IsNull() {
        updateReq.DHCPStop = plan.DHCPStop.ValueString()
    }

    // Update via API
    network, err := r.client.c.UpdateNetwork(ctx, site, updateReq)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error updating network",
            "Could not update network: "+err.Error(),
        )
        return
    }

    // Update state with API response
    r.updateModelFromAPI(ctx, &plan, network, site)

    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}

func (r *networkFrameworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state networkResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    site := state.Site.ValueString()
    if site == "" {
        site = r.client.site
    }

    err := r.client.c.DeleteNetwork(ctx, site, state.ID.ValueString(), state.Name.ValueString())
    if _, ok := err.(*unifi.NotFoundError); ok {
        // Resource already deleted
        return
    }
    if err != nil {
        resp.Diagnostics.AddError(
            "Error deleting network",
            "Could not delete network: "+err.Error(),
        )
        return
    }
}

func (r *networkFrameworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Support same import formats as SDK v2 version
    id := req.ID
    site := r.client.site

    // Handle site:id format
    if strings.Contains(id, ":") {
        parts := strings.SplitN(id, ":", 2)
        site = parts[0]
        id = parts[1]
    }

    // Handle name=NetworkName format
    if strings.HasPrefix(id, "name=") {
        targetName := strings.TrimPrefix(id, "name=")
        var err error
        // Note: getNetworkIDByName is an existing helper function in the codebase
        if id, err = getNetworkIDByName(ctx, r.client.c, targetName, site); err != nil {
            resp.Diagnostics.AddError(
                "Error finding network by name",
                "Could not find network with name "+targetName+": "+err.Error(),
            )
            return
        }
    }

    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
    resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
}
```

### 2. Register New Resources

Add the new Framework resources to `provider_framework.go`:

```go
func (p *frameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewNetworkFrameworkResource,
        // Add other Framework resources as they're migrated
    }
}
```

### 3. Testing Framework Resources

Create corresponding test files using Plugin Framework patterns:

```go
// internal/provider/resource_network_framework_test.go
func TestAccNetworkFramework_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        CheckDestroy:             testAccNetworkDestroy,
        Steps: []resource.TestStep{
            {
                Config: testAccNetworkFrameworkConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("unifi_network.test", "name", "test-network"),
                    resource.TestCheckResourceAttr("unifi_network.test", "purpose", "corporate"),
                ),
            },
            {
                ResourceName:      "unifi_network.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## Critical Migration Considerations

### 1. State Compatibility ⚠️

**Issue**: Plugin Framework handles state differently than SDK v2
**Solution**: Ensure consistent null/empty handling to prevent import issues

```go
// ❌ Wrong: Always setting values
model.Field = types.StringValue(apiResponse.Field)

// ✅ Correct: Handle empty strings as null
if apiResponse.Field == "" {
    model.Field = types.StringNull()
} else {
    model.Field = types.StringValue(apiResponse.Field)
}
```

### 2. Import Compatibility

Maintain backward compatibility with existing import formats:

```bash
# These should all continue to work
terraform import unifi_network.test 5dc28e5e9106d105bdc87217
terraform import unifi_network.test site1:5dc28e5e9106d105bdc87217  
terraform import unifi_network.test name=LAN
```

### 3. Validation Migration

Convert SDK v2 validation functions to Framework validators:

```go
// SDK v2
ValidateFunc: validation.StringInSlice([]string{"corporate", "guest", "wan"}, false)

// Framework
Validators: []validator.String{
    stringvalidator.OneOf("corporate", "guest", "wan"),
}
```

### 4. Type Conversion Utilities

Create utilities for consistent type conversion:

```go
// internal/provider/util/conversion.go
package util

import (
    "github.com/hashicorp/terraform-plugin-framework/types"
    "github.com/hashicorp/terraform-plugin-framework/attr"
)

// StringValueOrNull returns StringNull if empty, StringValue otherwise
func StringValueOrNull(s string) types.String {
    if s == "" {
        return types.StringNull()
    }
    return types.StringValue(s)
}

// Int64ValueOrNull returns Int64Null if zero, Int64Value otherwise  
func Int64ValueOrNull(i int64) types.Int64 {
    if i == 0 {
        return types.Int64Null()
    }
    return types.Int64Value(i)
}

// StringSliceToList converts []string to types.List, handling empty slices as null
func StringSliceToList(slice []string) types.List {
    if len(slice) == 0 {
        return types.ListNull(types.StringType)
    }
    
    values := make([]attr.Value, len(slice))
    for i, s := range slice {
        values[i] = types.StringValue(s)
    }
    
    list, _ := types.ListValue(types.StringType, values)
    return list
}
```

## Testing Strategy

### 1. Dual Testing During Migration

Run tests for both SDK v2 and Framework versions during the transition:

```bash
# Test existing SDK v2 resources
go test ./internal/provider -run TestAccNetwork_ -v

# Test new Framework resources  
go test ./internal/provider -run TestAccNetworkFramework_ -v
```

### 2. Import Testing

Ensure imports work correctly with Framework resources:

```bash
# Test various import formats
terraform import unifi_network.test 5dc28e5e9106d105bdc87217
terraform plan  # Should show no changes
```

### 3. State Migration Testing

Test that existing Terraform states work with new provider:

1. Create resource with SDK v2 provider
2. Upgrade to Framework provider
3. Run `terraform plan` - should show no changes

## Migration Checklist

### Infrastructure ✅
- [x] Plugin Framework dependencies added to `go.mod`
- [x] Mux server configured in `main.go`
- [x] Basic Framework provider structure in `provider_framework.go`
- [x] Test factories configured for Framework provider
- [ ] Add validator dependencies: `go get github.com/hashicorp/terraform-plugin-framework-validators`

### Per-Resource Migration
For each resource being migrated:

- [ ] Create Framework resource file (e.g., `resource_network_framework.go`)
- [ ] Implement schema with proper validation
- [ ] Implement CRUD operations with consistent null handling
- [ ] Implement import functionality matching SDK v2 behavior
- [ ] Add Framework resource to provider registration
- [ ] Create Framework-specific tests
- [ ] Verify import compatibility
- [ ] Verify no-op plans after import
- [ ] Performance testing

### Data Sources
- [ ] Migrate corresponding data sources using same patterns
- [ ] Ensure consistency between resources and data sources

### Documentation
- [ ] Update resource documentation for Framework-specific patterns
- [ ] Update import examples
- [ ] Update provider configuration examples

### Final Migration
- [ ] All resources migrated to Framework
- [ ] All data sources migrated to Framework  
- [ ] All tests passing
- [ ] Performance benchmarks acceptable
- [ ] Remove SDK v2 provider from mux
- [ ] Update `main.go` to Framework-only
- [ ] Remove unused SDK v2 dependencies

## Common Pitfalls & Solutions

### 1. Import State Mismatches
**Problem**: Plan shows changes after importing a resource
**Solution**: 
- Ensure consistent null handling in read functions
- Match computed field behavior between create and read
- Use utility functions for type conversion

### 2. Validation Conflicts  
**Problem**: Framework validators preventing valid configurations
**Solution**:
- Carefully review validation logic from SDK v2
- Test edge cases thoroughly
- Use proper conflict validators

### 3. Complex Nested Attributes
**Problem**: Framework requires explicit nested schemas
**Solution**:
- Define nested object schemas explicitly
- Avoid interface{} types
- Use proper type conversion utilities

### 4. Performance Issues
**Problem**: Framework resources performing slower than SDK v2
**Solution**:
- Profile resource operations
- Optimize API calls
- Consider batching operations where possible

## Resources for Reference

- [Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Migration Guide from SDK v2](https://developer.hashicorp.com/terraform/plugin/framework/migrating)
- [Framework Testing Guide](https://developer.hashicorp.com/terraform/plugin/framework/testing)
- [Muxing Documentation](https://developer.hashicorp.com/terraform/plugin/mux)
- [Ironic Provider Example](https://github.com/appkins-org/terraform-provider-ironic) - Complete migration reference

## Getting Help

When migrating resources:

1. Start with simple resources (fewer fields, simpler validation)
2. Use existing patterns from `provider_framework.go`
3. Leverage the type conversion utilities
4. Test import scenarios early and often
5. Refer to the [terraform-plugin-framework examples](https://github.com/hashicorp/terraform-plugin-framework/tree/main/examples)

This migration approach allows for gradual transition while maintaining backward compatibility through provider muxing until all resources are successfully migrated.