package unifi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
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
	*ui.ApiClient
	Site string
}

func New() provider.Provider {
	return &unifiProvider{}
}

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
		MarkdownDescription: "The UniFi provider is used to interact with UniFi Controller resources. " +
			"The provider needs to be configured with the proper credentials before it can be used.",

		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for the Unifi controller. Can be specified with the `UNIFI_API_KEY` " +
					"environment variable. If this is set, the `username` and `password` fields are ignored.",
				Optional:  true,
				Sensitive: true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Local user name for the Unifi controller API. Can be specified with the `UNIFI_USERNAME` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for the user accessing the API. Can be specified with the `UNIFI_PASSWORD` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: "URL of the controller API. Can be specified with the `UNIFI_API` environment variable. " +
					"You should **NOT** supply the path (`/api`), the SDK will discover the appropriate paths. This is " +
					"to support UDM Pro style API paths as well as more standard controller paths.",
				Optional: true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site in the Unifi controller this provider will manage. Can be specified with " +
					"the `UNIFI_SITE` environment variable. Default: `default`",
				Optional: true,
			},
			"allow_insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip verification of TLS certificates of API requests. You may need to set this to `true` " +
					"if you are using your local API without setting up a signed certificate. Can be specified with the " +
					"`UNIFI_INSECURE` environment variable. Ignored when `cloud_connector` is enabled.",
				Optional: true,
			},
			"cloud_connector": schema.BoolAttribute{
				MarkdownDescription: "Use UniFi Cloud Connector API to access the controller. When enabled, requires `api_key` " +
					"authentication and automatically routes requests through https://api.ui.com. Can be specified with the " +
					"`UNIFI_CLOUD_CONNECTOR` environment variable. The `api_url` field is ignored when this is enabled.",
				Optional: true,
			},
			"hardware_id": schema.StringAttribute{
				MarkdownDescription: "Hardware ID of the UniFi console to connect to when using Cloud Connector. " +
					"If not specified, defaults to the first console where owner=true. Can be specified with the " +
					"`UNIFI_HARDWARE_ID` environment variable. Only used when `cloud_connector` is enabled.",
				Optional: true,
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

	// Get values from configuration or environment variables
	apiUrl := config.ApiUrl.ValueString()
	if apiUrl == "" {
		if v := os.Getenv("UNIFI_API"); v != "" {
			apiUrl = v
		}
	}

	username := config.Username.ValueString()
	if username == "" {
		if v := os.Getenv("UNIFI_USERNAME"); v != "" {
			username = v
		}
	}

	password := config.Password.ValueString()
	if password == "" {
		if v := os.Getenv("UNIFI_PASSWORD"); v != "" {
			password = v
		}
	}

	apiKey := config.ApiKey.ValueString()
	if apiKey == "" {
		if v := os.Getenv("UNIFI_API_KEY"); v != "" {
			apiKey = v
		}
	}

	allowInsecure := config.AllowInsecure.ValueBool()
	if !allowInsecure {
		if v := os.Getenv("UNIFI_INSECURE"); v != "" {
			allowInsecure = v == "true"
		}
	}

	cloudConnector := config.CloudConnector.ValueBool()
	if !cloudConnector {
		if v := os.Getenv("UNIFI_CLOUD_CONNECTOR"); v != "" {
			cloudConnector = v == "true"
		}
	}

	hardwareID := config.HardwareID.ValueString()
	if hardwareID == "" {
		if v := os.Getenv("UNIFI_HARDWARE_ID"); v != "" {
			hardwareID = v
		}
	}

	site := config.Site.ValueString()
	if site == "" {
		if v := os.Getenv("UNIFI_SITE"); v != "" {
			site = v
		}
	}
	if site == "" {
		site = "default"
	}

	ctx = tflog.SetField(ctx, "unifi_api_url", apiUrl)
	ctx = tflog.SetField(ctx, "unifi_api_key", apiKey)
	ctx = tflog.SetField(ctx, "unifi_username", username)
	ctx = tflog.SetField(ctx, "unifi_password", password)
	ctx = tflog.SetField(ctx, "unifi_site", site)
	ctx = tflog.SetField(ctx, "unifi_allow_insecure", allowInsecure)
	ctx = tflog.SetField(ctx, "unifi_cloud_connector", cloudConnector)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "unifi_api_key")
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "unifi_password")

	tflog.Debug(ctx, "Creating Unifi client")

	// Validate required fields
	if cloudConnector {
		// Cloud Connector requires API key only
		if apiKey == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("api_key"),
				"Missing API Key for Cloud Connector",
				"Cloud Connector mode requires an API key. Username/password authentication is not supported.",
			)
		}
		// Force secure connections for cloud
		allowInsecure = false
	} else {
		// Direct connection validation
		if apiUrl == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("api_url"),
				"Missing API URL Configuration",
				"While configuring the provider, the API URL was not found in the configuration or UNIFI_API environment variable.",
			)
		}

		if apiKey == "" && (username == "" || password == "") {
			resp.Diagnostics.AddAttributeError(
				path.Root("api_key"),
				"Missing Authentication Configuration",
				"Either api_key or both username and password must be provided.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP client
	c := retryablehttp.NewClient()
	c.HTTPClient.Timeout = 30 * time.Second
	c.Logger = NewLogger(ctx)

	// Configure TLS if needed
	if allowInsecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
		c.HTTPClient.Transport = transport
	}

	// Add cookie jar for session management
	jar, _ := cookiejar.New(nil)
	c.HTTPClient.Jar = jar

	// Create UniFi client
	client := &ui.ApiClient{}
	if err := client.SetHTTPClient(c); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create HTTP Client",
			"An unexpected error occurred when creating the HTTP client. "+
				err.Error(),
		)
		return
	}

	// Set authentication (must be done before cloud connector setup)
	if apiKey != "" {
		client.SetAPIKey(apiKey)
	}

	// Configure Cloud Connector or Direct Connection
	if cloudConnector {
		// Enable Cloud Connector mode
		var consoleID string
		var err error

		if hardwareID != "" {
			// Use specific hardware ID
			consoleID, err = client.EnableCloudConnectorByHardwareID(ctx, hardwareID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Enabling Cloud Connector",
					fmt.Sprintf(
						"Could not find console with hardware ID %s: %s",
						hardwareID,
						err.Error(),
					),
				)
				return
			}
		} else {
			// Use default selection (first owner host)
			consoleID, err = client.EnableCloudConnector(ctx, -1)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Enabling Cloud Connector",
					"Could not enable Cloud Connector mode: "+err.Error(),
				)
				return
			}
		}

		// Log the selected console ID for debugging
		_ = consoleID // Console ID is now set in the client
	} else {
		// Direct connection mode
		if err := client.SetBaseURL(apiUrl); err != nil {
			resp.Diagnostics.AddError(
				"Invalid API URL",
				"The provided API URL is invalid. "+
					err.Error(),
			)
			return
		}

		if apiKey == "" && username != "" && password != "" {
			if err := client.Login(ctx, username, password); err != nil {
				resp.Diagnostics.AddError(
					"Error Logging In",
					"Could not log in with username and password. "+
						err.Error(),
				)
				return
			}
		}
	}

	// Create wrapper client with site info
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
		NewClientResource,
		NewClientGroupFrameworkResource,
		NewWANResource,
		NewWLANFrameworkResource,
		NewVirtualNetworkResource,
		NewVPNClientResource,
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

func (p *unifiProvider) EphemeralResources(
	ctx context.Context,
) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *unifiProvider) Actions(
	ctx context.Context,
) []func() action.Action {
	return []func() action.Action{
		NewPortAction,
	}
}

// ListResources implements [provider.ProviderWithListResources].
func (p *unifiProvider) ListResources(context.Context) []func() list.ListResource {
	return []func() list.ListResource{
		NewClientListResource,
	}
}
