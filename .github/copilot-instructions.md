# Terraform Provider Development Guide: Plugin Framework

## Overview

This guide provides comprehensive instructions for developing resources in the terraform-provider-unifi using the Terraform Plugin Framework. The provider is built entirely on the Plugin Framework (no SDK v2 dependencies).

## Prerequisites

- **Repository**: [terraform-provider-unifi](https://github.com/ubiquiti-community/terraform-provider-unifi)
- **Framework**: Terraform Plugin Framework v1.12+
- **Go Version**: 1.22+
- **API Client**: [go-unifi](https://github.com/ubiquiti-community/go-unifi)

## Key Documentation References

- [Plugin Framework Overview](https://developer.hashicorp.com/terraform/plugin/framework)
- [Provider Framework Tutorial](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework)
- [Schema Concepts](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/schemas)
- [Resource Implementation](https://developer.hashicorp.com/terraform/plugin/framework/resources)
- [Nested Attributes](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#nested-attribute-types)
- [Testing Framework](https://developer.hashicorp.com/terraform/plugin/framework/testing)

## Project Structure

The provider uses a flat structure within the `unifi/` directory:

```
terraform-provider-unifi/
├── main.go                    # Provider entry point
├── unifi/
│   ├── provider.go           # Provider implementation
│   ├── provider_test.go      # Provider tests
│   ├── resource_*.go         # Resource implementations
│   ├── data_source_*.go      # Data source implementations
│   ├── *_action.go           # Action implementations
│   └── util/                 # Utility functions
│       ├── conversion.go     # Type conversion helpers
│       └── retry/           # Retry logic
└── docs/                     # Generated documentation
```

## Provider Entry Point

The provider uses the Plugin Framework's server implementation:
```go
package main

import (
    "context"
    "flag"
    "log"

    "github.com/hashicorp/terraform-plugin-framework/providerserver"
    "github.com/ubiquiti-community/terraform-provider-unifi/unifi"
)

//go:generate go tool tfplugindocs generate -provider-name unifi
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

    err := providerserver.Serve(context.Background(), unifi.New, opts)
    if err != nil {
        log.Fatal(err.Error())
    }
}
```

## Provider Implementation

The provider implements multiple interfaces to support various features:

```go
package unifi

import (
    "github.com/hashicorp/terraform-plugin-framework/action"
    "github.com/hashicorp/terraform-plugin-framework/datasource"
    "github.com/hashicorp/terraform-plugin-framework/ephemeral"
    "github.com/hashicorp/terraform-plugin-framework/list"
    "github.com/hashicorp/terraform-plugin-framework/provider"
    "github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
    _ provider.Provider                       = &unifiProvider{}
    _ provider.ProviderWithEphemeralResources = &unifiProvider{}
    _ provider.ProviderWithListResources      = &unifiProvider{}
)

type unifiProvider struct{}

type unifiProviderModel struct {
    ApiKey         types.String `tfsdk:"api_key"`
    Username       types.String `tfsdk:"username"`
    Password       types.String `tfsdk:"password"`
    ApiUrl         types.String `tfsdk:"api_url"`
    Site           types.String `tfsdk:"site"`
    AllowInsecure  types.Bool   `tfsdk:"allow_insecure"`
    CloudConnector types.Bool   `tfsdk:"cloud_connector"`
    HardwareID     types.String `tfsdk:"hardware_id"`
}

// Client wraps the UniFi client with site information.
type Client struct {
    *unifi.ApiClient
    Site string
}

func New() provider.Provider {
    return &unifiProvider{}
}
```

### Provider Methods

```go
func (p *unifiProvider) Metadata(
    ctx context.Context,
    req provider.MetadataRequest,
    resp *provider.MetadataResponse,
) {
    resp.TypeName = "unifi"
}

func (p *unifiProvider) Schema(
    ctx context.Context,
    req provider.SchemaRequest,
    resp *provider.SchemaResponse,
) {
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
            "api_key": schema.StringAttribute{
                Optional:    true,
                Sensitive:   true,
                Description: "API key for UniFi controller (cloud connector)",
            },
            "allow_insecure": schema.BoolAttribute{
                Optional:    true,
                Description: "Allow insecure TLS connections",
            },
            "site": schema.StringAttribute{
                Optional:    true,
                Description: "UniFi site name (defaults to 'default')",
            },
            "cloud_connector": schema.BoolAttribute{
                Optional:    true,
                Description: "Use UniFi Cloud Connector authentication",
            },
            "hardware_id": schema.StringAttribute{
                Optional:    true,
                Description: "Hardware ID for cloud connector authentication",
            },
        },
    }
}

func (p *unifiProvider) Configure(
    ctx context.Context,
    req provider.ConfigureRequest,
    resp *provider.ConfigureResponse,
) {
    var config unifiProviderModel
    diags := req.Config.Get(ctx, &config)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Read configuration from environment variables or config
    // Create HTTP client with proper TLS settings
    // Authenticate with UniFi controller
    // Store configured client in provider data
    
    configuredClient := &Client{
        ApiClient: client,
        Site:      site,
    }

    resp.DataSourceData = configuredClient
    resp.ResourceData = configuredClient
    resp.EphemeralResourceData = configuredClient
    resp.ActionData = configuredClient
    resp.ListResourceData = configuredClient
}

func (p *unifiProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource{
        NewAccountFrameworkResource,
        NewBGPResource,
        NewClientResource,
        NewDeviceFrameworkResource,
        NewDNSRecordFrameworkResource,
        NewDynamicDNSResource,
        NewFirewallGroupFrameworkResource,
        NewFirewallRuleResource,
        NewNetworkResource,
        NewPortForwardResource,
        NewPortProfileFrameworkResource,
        NewRadiusProfileResource,
        NewSettingResource,
        NewSiteFrameworkResource,
        NewStaticRouteFrameworkResource,
        NewClientGroupFrameworkResource,
        NewWANResource,
        NewWLANFrameworkResource,
        NewVirtualNetworkResource,
    }
}

func (p *unifiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource{
        NewClientDataSource,
        NewClientInfoDataSource,
        NewClientInfoListDataSource,
        NewNetworkDataSource,
        NewAccountDataSource,
        NewAPGroupDataSource,
        NewDNSRecordDataSource,
        NewPortProfileDataSource,
        NewRadiusProfileDataSource,
        NewClientGroupDataSource,
    }
}

func (p *unifiProvider) ListResources(ctx context.Context) []func() list.ListResource {
    return []func() list.ListResource{
        NewClientListResource,
    }
}

func (p *unifiProvider) Actions(ctx context.Context) []func() action.Action {
    return []func() action.Action{
        NewPortAction,
    }
}

func (p *unifiProvider) EphemeralResources(
    ctx context.Context,
) []func() ephemeral.EphemeralResource {
    return []func() ephemeral.EphemeralResource{}
}
```

## Resource Implementation Pattern

Resources follow a consistent pattern in this provider:

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

## Nested Objects and Complex Schemas

### Best Practices for Nested Attributes

The Plugin Framework provides nested attribute types that are preferred over object attributes:

- **`SingleNestedAttribute`**: For a single object with defined attributes
- **`ListNestedAttribute`**: For an ordered collection of nested objects
- **`SetNestedAttribute`**: For an unordered, unique collection of nested objects
- **`MapNestedAttribute`**: For a mapping of string keys to nested objects

**Key Points:**
- Use nested attribute types instead of `ObjectAttribute` for better control
- Each nested attribute must define its own configurability (`Required`, `Optional`, `Computed`)
- The `NestedObject` field defines the schema of nested elements
- Nested attributes support validation, plan modification, and description at each level

### Example: Virtual Network with Nested DHCP Configuration

See [virtual_network_resource.go](../unifi/virtual_network_resource.go) for a comprehensive example of nested objects.

#### Schema Definition

```go
type dhcpBootModel struct {
    Enabled  types.Bool   `tfsdk:"enabled"`
    Server   types.String `tfsdk:"server"`
    Filename types.String `tfsdk:"filename"`
}

func (m dhcpBootModel) AttributeTypes() map[string]attr.Type {
    return map[string]attr.Type{
        "enabled":  types.BoolType,
        "server":   types.StringType,
        "filename": types.StringType,
    }
}

type dhcpServerModel struct {
    Boot           types.Object `tfsdk:"boot"`
    Enabled        types.Bool   `tfsdk:"enabled"`
    Start          types.String `tfsdk:"start"`
    Stop           types.String `tfsdk:"stop"`
    GatewayEnabled types.Bool   `tfsdk:"gateway_enabled"`
    DnsEnabled     types.Bool   `tfsdk:"dns_enabled"`
    DnsServers     types.List   `tfsdk:"dns_servers"`
}

func (m dhcpServerModel) AttributeTypes() map[string]attr.Type {
    return map[string]attr.Type{
        "boot":            types.ObjectType{AttrTypes: dhcpBootModel{}.AttributeTypes()},
        "enabled":         types.BoolType,
        "start":           types.StringType,
        "stop":            types.StringType,
        "gateway_enabled": types.BoolType,
        "dns_enabled":     types.BoolType,
        "dns_servers":     types.ListType{ElemType: types.StringType},
    }
}

type virtualNetworkResourceModel struct {
    ID         types.String `tfsdk:"id"`
    Site       types.String `tfsdk:"site"`
    Name       types.String `tfsdk:"name"`
    Subnet     types.String `tfsdk:"subnet"`
    DhcpServer types.Object `tfsdk:"dhcp_server"`
    DhcpRelay  types.Object `tfsdk:"dhcp_relay"`
}

// In Schema method:
resp.Schema = schema.Schema{
    Attributes: map[string]schema.Attribute{
        "dhcp_server": schema.SingleNestedAttribute{
            Optional:            true,
            MarkdownDescription: "DHCP server configuration",
            Attributes: map[string]schema.Attribute{
                "boot": schema.SingleNestedAttribute{
                    Optional: true,
                    Attributes: map[string]schema.Attribute{
                        "enabled": schema.BoolAttribute{
                            Optional: true,
                        },
                        "server": schema.StringAttribute{
                            Optional: true,
                        },
                        "filename": schema.StringAttribute{
                            Optional: true,
                        },
                    },
                },
                "enabled": schema.BoolAttribute{
                    Required: true,
                },
                "start": schema.StringAttribute{
                    Optional: true,
                },
                "stop": schema.StringAttribute{
                    Optional: true,
                },
                "dns_servers": schema.ListAttribute{
                    Optional:    true,
                    ElementType: types.StringType,
                },
            },
        },
    },
}
```

#### Mapping Nested Objects to API Prefixed Fields

The UniFi API often uses flat structures with prefixed fields (e.g., `dhcpd_*`, `dhcpd_boot_*`). 
Nested Terraform objects provide a better user experience:

**Terraform Configuration:**
```hcl
resource "unifi_virtual_network" "example" {
  name   = "example"
  subnet = "10.0.0.0/24"
  
  dhcp_server = {
    enabled = true
    start   = "10.0.0.100"
    stop    = "10.0.0.200"
    
    boot = {
      enabled  = true
      server   = "10.0.0.1"
      filename = "boot.cfg"
    }
    
    dns_servers = ["8.8.8.8", "8.8.4.4"]
  }
}
```

**Conversion to API Model:**

```go
func modelToNetwork(
    ctx context.Context,
    model *virtualNetworkResourceModel,
) (*unifi.Network, diag.Diagnostics) {
    var diags diag.Diagnostics
    
    network := &unifi.Network{
        Name:   model.Name.ValueString(),
        Subnet: model.Subnet.ValueString(),
    }
    
    // Handle DHCP server configuration
    if !model.DhcpServer.IsNull() && !model.DhcpServer.IsUnknown() {
        var dhcpServer dhcpServerModel
        d := model.DhcpServer.As(ctx, &dhcpServer, basetypes.ObjectAsOptions{})
        diags.Append(d...)
        if !diags.HasError() {
            // Map dhcp_server.enabled -> DHCPDEnabled
            network.DHCPDEnabled = dhcpServer.Enabled.ValueBool()
            network.DHCPDStart = dhcpServer.Start.ValueStringPointer()
            network.DHCPDStop = dhcpServer.Stop.ValueStringPointer()
            network.DHCPDDNSEnabled = dhcpServer.DnsEnabled.ValueBool()
            
            // Handle nested boot configuration
            if !dhcpServer.Boot.IsNull() && !dhcpServer.Boot.IsUnknown() {
                var dhcpBoot dhcpBootModel
                d := dhcpServer.Boot.As(ctx, &dhcpBoot, basetypes.ObjectAsOptions{})
                diags.Append(d...)
                if !diags.HasError() {
                    // Map dhcp_server.boot.* -> DHCPDBoot*
                    network.DHCPDBootEnabled = dhcpBoot.Enabled.ValueBool()
                    network.DHCPDBootServer = dhcpBoot.Server.ValueString()
                    network.DHCPDBootFilename = dhcpBoot.Filename.ValueStringPointer()
                }
            }
            
            // Handle DNS servers list
            if !dhcpServer.DnsServers.IsNull() && !dhcpServer.DnsServers.IsUnknown() {
                var dnsServers []string
                d := dhcpServer.DnsServers.ElementsAs(ctx, &dnsServers, false)
                diags.Append(d...)
                if !diags.HasError() {
                    // API uses DHCPDDNS1, DHCPDDNS2, DHCPDDNS3, DHCPDDNS4
                    for i, dns := range dnsServers {
                        if i >= 4 {
                            break
                        }
                        switch i {
                        case 0:
                            network.DHCPDDNS1 = dns
                        case 1:
                            network.DHCPDDNS2 = dns
                        case 2:
                            network.DHCPDDNS3 = dns
                        case 3:
                            network.DHCPDDNS4 = dns
                        }
                    }
                }
            }
        }
    }
    
    return network, diags
}
```

**Conversion from API Model:**

```go
func networkToModel(
    ctx context.Context,
    network *unifi.Network,
    model *virtualNetworkResourceModel,
    site string,
    previousModel *virtualNetworkResourceModel,
) diag.Diagnostics {
    var diags diag.Diagnostics
    
    model.ID = types.StringValue(network.ID)
    model.Site = types.StringValue(site)
    model.Name = types.StringValue(network.Name)
    model.Subnet = types.StringValue(network.Subnet)
    
    // Only populate dhcp_server if it was configured in previous state
    // or if this is an import and DHCP is enabled
    shouldPopulateDhcp := !previousModel.DhcpServer.IsNull() || network.DHCPDEnabled
    
    if shouldPopulateDhcp {
        // Create nested boot object if boot is enabled
        var bootObj types.Object
        if network.DHCPDBootEnabled {
            bootModel := dhcpBootModel{
                Enabled:  types.BoolValue(network.DHCPDBootEnabled),
                Server:   types.StringValue(network.DHCPDBootServer),
                Filename: types.StringPointerValue(network.DHCPDBootFilename),
            }
            var d diag.Diagnostics
            bootObj, d = types.ObjectValueFrom(ctx, dhcpBootModel{}.AttributeTypes(), bootModel)
            diags.Append(d...)
        } else {
            bootObj = types.ObjectNull(dhcpBootModel{}.AttributeTypes())
        }
        
        // Collect DNS servers into list
        var dnsServersList types.List
        dnsServers := []string{}
        if network.DHCPDDNS1 != "" {
            dnsServers = append(dnsServers, network.DHCPDDNS1)
        }
        if network.DHCPDDNS2 != "" {
            dnsServers = append(dnsServers, network.DHCPDDNS2)
        }
        if len(dnsServers) > 0 {
            var d diag.Diagnostics
            dnsServersList, d = types.ListValueFrom(ctx, types.StringType, dnsServers)
            diags.Append(d...)
        } else {
            dnsServersList = types.ListNull(types.StringType)
        }
        
        // Create DHCP server object
        dhcpServerModel := dhcpServerModel{
            Boot:           bootObj,
            Enabled:        types.BoolValue(network.DHCPDEnabled),
            Start:          types.StringPointerValue(network.DHCPDStart),
            Stop:           types.StringPointerValue(network.DHCPDStop),
            GatewayEnabled: types.BoolValue(network.DHCPDGatewayEnabled),
            DnsEnabled:     types.BoolValue(network.DHCPDDNSEnabled),
            DnsServers:     dnsServersList,
        }
        
        var d diag.Diagnostics
        model.DhcpServer, d = types.ObjectValueFrom(ctx, dhcpServerModel.AttributeTypes(), dhcpServerModel)
        diags.Append(d...)
    } else {
        model.DhcpServer = types.ObjectNull(dhcpServerModel{}.AttributeTypes())
    }
    
    return diags
}
```

### Key Patterns for Nested Objects

1. **Define Helper Models**: Create structs for each nested object level
2. **Implement AttributeTypes()**: Return the type map for each model
3. **Use SingleNestedAttribute**: In schema definition with nested Attributes map
4. **Handle Null/Unknown**: Always check before accessing nested values
5. **Preserve State**: Only populate nested objects that were configured or during import
6. **Map to API Prefixes**: Convert nested structure to flat API fields with consistent prefixes
7. **Convert from API**: Reverse the mapping, grouping prefixed fields back into nested objects

## Type Conversion Best Practices

### Handling Null vs Empty Values

```go
// ❌ Wrong: Always setting values
model.LastError = types.StringValue(apiResponse.LastError)

// ✅ Correct: Handle empty strings as null
if apiResponse.LastError == "" {
    model.LastError = types.StringNull()
} else {
    model.LastError = types.StringValue(apiResponse.LastError)
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

### Pointer Handling

```go
// ValueStringPointer: Returns nil if types.String is null
network.DHCPDStart = dhcpServer.Start.ValueStringPointer()

// StringPointerValue: Creates types.StringNull if pointer is nil
model.Start = types.StringPointerValue(network.DHCPDStart)
```

## Testing

### Provider Factory Setup

In `provider_test.go`:

```go
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
    "unifi": providerserver.NewProtocol6WithError(New()),
}

func testAccPreCheck(t *testing.T) {
    if os.Getenv("UNIFI_API_URL") == "" {
        t.Fatal("UNIFI_API_URL must be set for acceptance tests")
    }
}
```

### Acceptance Tests

```go
func TestAccWLAN_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccWLANConfig_basic(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("unifi_wlan.test", "name", "test-network"),
                    resource.TestCheckResourceAttr("unifi_wlan.test", "ssid", "test-ssid"),
                ),
            },
            {
                ResourceName:            "unifi_wlan.test",
                ImportState:             true,
                ImportStateVerify:       true,
                ImportStateVerifyIgnore: []string{"passphrase"}, // Sensitive fields
            },
        },
    })
}

func testAccWLANConfig_basic() string {
    return `
resource "unifi_wlan" "test" {
  name       = "test-network"
  ssid       = "test-ssid"
  security   = "wpapsk"
  passphrase = "testpassword123"
  vlan_id    = 10
}
`
}
```

## Data Source Implementation

Data sources follow similar patterns to resources but are read-only:

```go
type wlanDataSource struct {
    client *Client
}

func (d *wlanDataSource) Read(
    ctx context.Context,
    req datasource.ReadRequest,
    resp *datasource.ReadResponse,
) {
    var config wlanDataSourceModel
    diags := req.Config.Get(ctx, &config)
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Query API based on config parameters
    // Populate state with results
    
    diags = resp.State.Set(ctx, &state)
    resp.Diagnostics.Append(diags...)
}
```

## Import Functionality

Support both simple and complex import formats:

```go
func (r *wlanFrameworkResource) ImportState(
    ctx context.Context,
    req resource.ImportStateRequest,
    resp *resource.ImportStateResponse,
) {
    // Support formats: "id" or "site:id"
    idParts := strings.Split(req.ID, ":")
    if len(idParts) == 2 {
        resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
        resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
    } else {
        resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
    }
}
```

For complex identifiers (like MAC addresses):

```go
func (r *clientResource) ImportState(
    ctx context.Context,
    req resource.ImportStateRequest,
    resp *resource.ImportStateResponse,
) {
    // Try to parse as MAC address
    _, err := net.ParseMAC(req.ID)
    if err == nil {
        // It's a MAC address
        resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mac"), req.ID)...)
    } else {
        // Regular ID or site:id format
        idParts := strings.Split(req.ID, ":")
        if len(idParts) == 2 {
            resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), idParts[0])...)
            resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
        } else {
            resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
        }
    }
}
```

## Common Pitfalls & Solutions

### 1. Schema Definition
- **Issue**: Forgetting to mark computed attributes
- **Solution**: Attributes set by API (like ID, timestamps) must be `Computed: true`

### 2. Client Configuration
- **Issue**: Nil client in resource methods
- **Solution**: Always check `req.ProviderData == nil` in Configure method

### 3. State Management
- **Issue**: Framework handles null/unknown differently than expected
- **Solution**: Always check `IsNull()` and `IsUnknown()` before operations

### 4. Import State Mismatches
- **Issue**: Plan shows changes after importing a resource
- **Solution**: Ensure Read method handles null values consistently with Create/Update

### 5. Nested Object Complexity
- **Issue**: Difficult to maintain deeply nested objects
- **Solution**: Limit nesting to 2-3 levels; use helper functions for conversions

## Validation

Use built-in validators for common patterns:

```go
import (
    "github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
    "github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

"security": schema.StringAttribute{
    Optional: true,
    Validators: []validator.String{
        stringvalidator.OneOf("open", "wep", "wpapsk", "wpaeap", "wpa3"),
    },
},
"vlan_id": schema.Int64Attribute{
    Optional: true,
    Validators: []validator.Int64{
        int64validator.Between(1, 4094),
    },
},
```

## Validation Checklist

- [ ] All resources implement required interfaces
- [ ] Import functionality works correctly
- [ ] Tests pass with acceptance test suite
- [ ] Documentation generated with tfplugindocs
- [ ] No-op plans after apply/import
- [ ] Error handling follows Framework patterns
- [ ] Nested objects map correctly to API fields
- [ ] Null/empty values handled consistently

## Resources for Reference

- [Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Framework Testing](https://developer.hashicorp.com/terraform/plugin/framework/testing)
- [go-unifi API Client](https://github.com/ubiquiti-community/go-unifi)
