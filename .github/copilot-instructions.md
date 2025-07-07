# Terraform Provider Migration: SDK v2 to Plugin Framework

## Overview

This guide provides detailed instructions for migrating a Terraform provider from SDK v2 to the newer Plugin Framework, based on real-world experience migrating terraform-provider-ironic and recommendations for terraform-provider-unifi.

## Prerequisites

- **Target Repository**: [terraform-provider-unifi (terraform-provider-mux branch)](https://github.com/ubiquiti-community/terraform-provider-unifi/tree/terraform-provider-mux)
- **Framework**: Terraform Plugin Framework v1.0+
- **Go Version**: 1.19+

## Key Documentation References

- [Plugin Framework Overview](https://developer.hashicorp.com/terraform/plugin/framework)
- [Migration Guide from SDK v2](https://developer.hashicorp.com/terraform/plugin/framework/migrating)
- [Provider Framework Tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework)
- [Schema Concepts](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas)
- [Resource Implementation](https://developer.hashicorp.com/terraform/plugin/framework/resources)

## Migration Steps

### 1. Project Structure Reorganization

**Current Structure** (SDK v2):
```
internal/
├── provider/
├── resources/
└── datasources/
```

**Target Structure** (Plugin Framework):
```
{provider_name}/          # Use 'unifi' for terraform-provider-unifi
├── provider.go
├── resource_*.go
├── data_source_*.go
└── util/
    └── conversion.go
```

### 2. Update Dependencies

Update `go.mod`:
```go
module github.com/ubiquiti-community/terraform-provider-unifi

require (
    github.com/hashicorp/terraform-plugin-framework v1.4.2
    github.com/hashicorp/terraform-plugin-framework-validators v0.12.0
    github.com/hashicorp/terraform-plugin-go v0.19.0
    github.com/hashicorp/terraform-plugin-mux v0.12.0
    // Keep existing dependencies for SDK v2 resources during transition
)
```

### 3. Main Provider Entry Point

**For Framework-Only Provider** (like ironic):
```go
package main

import (
    "context"
    "flag"
    "log"

    provider "github.com/ubiquiti-community/terraform-provider-unifi/unifi"
    "github.com/hashicorp/terraform-plugin-framework/providerserver"
)

//go:generate go tool github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name unifi
func main() {
    var debug bool

    flag.BoolVar(
        &debug,
        "debug",
        false,
        "set to true to run the provider with support for debuggers like delve",
    )
    flag.Parse()

    opts := providerserver.ServeOpts{
        Address: "registry.terraform.io/ubiquiti-community/unifi",
        Debug:   debug,
    }

    err := providerserver.Serve(context.Background(), provider.New(), opts)
    if err != nil {
        log.Fatal(err.Error())
    }
}
```

**For Mixed Provider** (SDK v2 + Framework):
```go
package main

import (
    "context"
    "flag"
    "log"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/hashicorp/terraform-plugin-mux/tf5to6server"
    "github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
    
    // SDK v2 provider (existing)
    sdkProvider "github.com/ubiquiti-community/terraform-provider-unifi/internal/provider"
    // Plugin Framework provider (new)
    frameworkProvider "github.com/ubiquiti-community/terraform-provider-unifi/unifi"
)

func main() {
    var debug bool
    flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers")
    flag.Parse()

    ctx := context.Background()

    // Upgrade SDK v2 provider to protocol version 6
    upgradedSdkProvider, err := tf5to6server.UpgradeServer(
        ctx,
        sdkProvider.New().GRPCProvider,
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create muxed provider
    muxServer, err := tf6muxserver.NewMuxServer(ctx,
        upgradedSdkProvider,
        providerserver.NewProtocol6(frameworkProvider.New()),
    )
    if err != nil {
        log.Fatal(err)
    }

    err = muxServer.Serve(ctx, &tf6muxserver.ServeOpts{
        Address: "registry.terraform.io/ubiquiti-community/unifi",
        Debug:   debug,
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### 4. Provider Implementation

Create `unifi/provider.go`:
```go
package unifi

import (
    "context"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/provider/schema"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

type unifiProvider struct {
    version string
}

type unifiProviderModel struct {
    Username      types.String `tfsdk:"username"`
    Password      types.String `tfsdk:"password"`
    APIUrl        types.String `tfsdk:"api_url"`
    AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
}

func New() provider.Provider {
    return &unifiProvider{}
}

func (p *unifiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "username": schema.StringAttribute{
                Optional:    true,
                Description: "Username for UniFi controller",
            },
            "password": schema.StringAttribute{
                Optional:    true,
                Sensitive:   true,
                Description: "Password for UniFi controller",
            },
            "api_url": schema.StringAttribute{
                Optional:    true,
                Description: "URL of the UniFi controller",
            },
            "allow_insecure": schema.BoolAttribute{
                Optional:    true,
                Description: "Allow insecure connections",
            },
        },
    }
}

func (p *unifiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var config unifiProviderModel
    diags := req.Config.Get(ctx, &config)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Configure your UniFi client here
    // Store client in resp.DataSourceData and resp.ResourceData
}

func (p *unifiProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        // Add your new framework resources here
        // NewWifiNetworkResource,
    }
}

func (p *unifiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        // Add your new framework data sources here
    }
}

func (p *unifiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "unifi"
    resp.Version = p.version
}
```

### 5. Resource Migration Pattern

For each SDK v2 resource, create a new Plugin Framework version:

**Example: WiFi Network Resource** (`unifi/resource_wifi_network.go`):
```go
package unifi

import (
    "context"
    "github.com/hashicorp/terraform-plugin-framework/resource"
    "github.com/hashicorp/terraform-plugin-framework/resource/schema"
    "github.com/hashicorp/terraform-plugin-framework/types"
)

type wifiNetworkResource struct {
    client *unifi.Client // Your UniFi client
}

type wifiNetworkResourceModel struct {
    ID       types.String `tfsdk:"id"`
    Name     types.String `tfsdk:"name"`
    SSID     types.String `tfsdk:"ssid"`
    Security types.String `tfsdk:"security"`
    Password types.String `tfsdk:"password"`
    VLANId   types.Int64  `tfsdk:"vlan_id"`
}

func NewWifiNetworkResource() resource.Resource {
    return &wifiNetworkResource{}
}

func (r *wifiNetworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a WiFi network in UniFi controller",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed:    true,
                Description: "The ID of the WiFi network",
            },
            "name": schema.StringAttribute{
                Required:    true,
                Description: "The name of the WiFi network",
            },
            "ssid": schema.StringAttribute{
                Required:    true,
                Description: "The SSID of the WiFi network",
            },
            "security": schema.StringAttribute{
                Optional:    true,
                Computed:    true,
                Description: "Security type (open, wep, wpa, wpa2, wpa3)",
            },
            "password": schema.StringAttribute{
                Optional:    true,
                Sensitive:   true,
                Description: "WiFi password",
            },
            "vlan_id": schema.Int64Attribute{
                Optional:    true,
                Description: "VLAN ID for the network",
            },
        },
    }
}

// Implement CRUD operations...
func (r *wifiNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan wifiNetworkResourceModel
    diags := req.Plan.Get(ctx, &plan)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Create the resource via UniFi API
    // Handle the response and update the plan
    
    diags = resp.State.Set(ctx, plan)
    resp.Diagnostics.Append(diags...)
}

// Implement Read, Update, Delete, ImportState methods...
func (r *wifiNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state wifiNetworkResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Read from API and update state
    // Critical: Handle null vs empty values consistently
    if apiResponse.Field == "" {
        state.Field = types.StringNull()
    } else {
        state.Field = types.StringValue(apiResponse.Field)
    }

    diags = resp.State.Set(ctx, state)
    resp.Diagnostics.Append(diags...)
}

func (r *wifiNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // Implementation similar to Create
}

func (r *wifiNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var state wifiNetworkResourceModel
    diags := req.State.Get(ctx, &state)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Delete via API
}

func (r *wifiNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *wifiNetworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_wifi_network"
}

func (r *wifiNetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    client, ok := req.ProviderData.(*unifi.Client)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            "Expected *unifi.Client, got something else",
        )
        return
    }

    r.client = client
}
```

### 6. Data Source Migration

Create corresponding data sources following similar patterns to resources.

### 7. Testing Migration

Update tests to use the new test patterns:
```go
func TestAccWifiNetwork_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccWifiNetworkConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("unifi_wifi_network.test", "name", "test-network"),
                ),
            },
            {
                ResourceName:      "unifi_wifi_network.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## Critical Migration Considerations

### 1. State Compatibility
- **Issue**: Plugin Framework handles state differently than SDK v2
- **Solution**: Implement proper state upgrade logic if needed
- **Test**: Ensure existing Terraform states work with new provider

### 2. Type Conversion Patterns
Based on ironic provider experience:
```go
// Handle null vs empty consistently
if apiResponse.Field == "" {
    model.Field = types.StringNull()
} else {
    model.Field = types.StringValue(apiResponse.Field)
}

// Lists should be null when empty
if len(apiResponse.Items) > 0 {
    itemValues := make([]attr.Value, len(apiResponse.Items))
    for i, item := range apiResponse.Items {
        itemValues[i] = types.StringValue(item)
    }
    model.Items, diags = types.ListValue(types.StringType, itemValues)
} else {
    model.Items = types.ListNull(types.StringType)
}
```

### 3. Import Functionality
- Ensure no-op plans after import
- Handle computed vs configured attributes properly
- Test import scenarios thoroughly

### 4. Validation Migration
```go
// SDK v2
validateFunc: validation.StringInSlice([]string{"option1", "option2"}, false)

// Plugin Framework
Validators: []validator.String{
    stringvalidator.OneOf("option1", "option2"),
}
```

## UniFi-Specific Migration Strategy

### Phase 1: Infrastructure Setup
1. Update `terraform-provider-mux` branch with Plugin Framework provider
2. Set up muxing in `main.go`
3. Create basic provider structure in `unifi/` directory

### Phase 2: Core Resource Migration
Prioritize these resources for initial migration:
1. `unifi_wifi_network` - Core WiFi network management
2. `unifi_user` - User management
3. `unifi_port_profile` - Port profile configuration

### Phase 3: Advanced Features
1. Complex nested resources
2. Advanced validation patterns
3. Custom plan modifiers

### Phase 4: Testing & Documentation
1. Comprehensive acceptance tests
2. Import testing
3. Documentation updates

## Common Pitfalls & Solutions

### 1. Schema Definition Differences
- **Issue**: Attribute syntax differs between SDK v2 and Framework
- **Solution**: Use schema validation tools and reference examples

### 2. Client Configuration
- **Issue**: Provider configuration handling changes
- **Solution**: Store client in provider data, access in resources via `req.ProviderData`

### 3. State Management
- **Issue**: Framework handles null/unknown differently
- **Solution**: Always check for null/unknown before operations, use consistent patterns

### 4. Complex Nested Attributes
- **Issue**: Framework requires explicit nested attribute schemas
- **Solution**: Define nested object schemas explicitly, avoid interface{} types

### 5. Import State Mismatches
- **Issue**: Plan shows changes after importing a resource
- **Solution**: Ensure consistent null handling in read functions
```go
// ❌ Wrong: Always setting values
model.LastError = types.StringValue(apiResponse.LastError)

// ✅ Correct: Handle empty strings as null
if apiResponse.LastError == "" {
    model.LastError = types.StringNull()
} else {
    model.LastError = types.StringValue(apiResponse.LastError)
}
```

## Validation Checklist

- [ ] All existing resources have Framework equivalents
- [ ] State compatibility maintained
- [ ] Import functionality works
- [ ] Tests pass with both providers
- [ ] Documentation updated
- [ ] No-op plans after apply/import
- [ ] Error handling follows Framework patterns
- [ ] Performance impact assessed

## Resources for Reference

- [Ironic Provider Example](https://github.com/appkins-org/terraform-provider-ironic) - Complete migration example
- [Framework Migration Guide](https://developer.hashicorp.com/terraform/plugin/framework/migrating)
- [Muxing Documentation](https://developer.hashicorp.com/terraform/plugin/mux)
- [Framework Testing](https://developer.hashicorp.com/terraform/plugin/framework/testing)

This migration approach allows for gradual transition while maintaining backward compatibility through provider muxing.
