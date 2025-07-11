package unifi

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ provider.Provider                       = &unifiProvider{}
	_ provider.ProviderWithEphemeralResources = &unifiProvider{}
)

type unifiProvider struct {
	version string
}

type unifiProviderModel struct {
	ApiKey        types.String `tfsdk:"api_key"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	ApiUrl        types.String `tfsdk:"api_url"`
	Site          types.String `tfsdk:"site"`
	AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
}

// Client wraps the UniFi client with site information.
type Client struct {
	*ui.Client
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
				Required: true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The site in the Unifi controller this provider will manage. Can be specified with " +
					"the `UNIFI_SITE` environment variable. Default: `default`",
				Optional: true,
			},
			"allow_insecure": schema.BoolAttribute{
				MarkdownDescription: "Skip verification of TLS certificates of API requests. You may need to set this to `true` " +
					"if you are using your local API without setting up a signed certificate. Can be specified with the " +
					"`UNIFI_INSECURE` environment variable.",
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

	site := config.Site.ValueString()
	if site == "" {
		if v := os.Getenv("UNIFI_SITE"); v != "" {
			site = v
		}
	}
	if site == "" {
		site = "default"
	}

	// Validate required fields
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

	if resp.Diagnostics.HasError() {
		return
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Configure TLS if needed
	if allowInsecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
		httpClient.Transport = transport
	}

	// Add cookie jar for session management
	jar, _ := cookiejar.New(nil)
	httpClient.Jar = jar

	// Create UniFi client
	client := &ui.Client{}
	if err := client.SetHTTPClient(httpClient); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create HTTP Client",
			"An unexpected error occurred when creating the HTTP client. "+
				"UniFi Client Error: "+err.Error(),
		)
		return
	}

	if err := client.SetBaseURL(apiUrl); err != nil {
		resp.Diagnostics.AddError(
			"Invalid API URL",
			"The provided API URL is invalid. "+
				"UniFi Client Error: "+err.Error(),
		)
		return
	}

	// Set authentication
	if apiKey != "" {
		client.SetAPIKey(apiKey)
	} else if username != "" && password != "" {
		if err := client.Login(ctx, username, password); err != nil {
			resp.Diagnostics.AddError(
				"Error Logging In",
				"Could not log in with username and password. "+
					"UniFi Client Error: "+err.Error(),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Authentication Configuration",
			"Either api_key or both username and password must be provided.",
		)
		return
	}

	// Create wrapper client with site info
	configuredClient := &Client{
		Client: client,
		Site:   site,
	}

	if err := configuredClient.Login(ctx, username, password); err != nil {
		resp.Diagnostics.AddError(
			"Error Logging In",
			"Could not log in with username and password. "+
				"UniFi Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = configuredClient
	resp.ResourceData = configuredClient
	resp.EphemeralResourceData = configuredClient
}

func (p *unifiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAccountFrameworkResource,
		NewDeviceFrameworkResource,
		NewDNSRecordFrameworkResource,
		NewDynamicDNSResource,
		NewFirewallGroupFrameworkResource,
		NewFirewallRuleResource,
		NewNetworkResource,
		NewPortForwardResource,
		NewPortProfileFrameworkResource,
		NewRadiusProfileResource,
		NewSettingMgmtResource,
		NewSettingRadiusResource,
		NewSettingUSGResource,
		NewSiteFrameworkResource,
		NewStaticRouteFrameworkResource,
		NewUserFrameworkResource,
		NewUserGroupFrameworkResource,
		NewWLANFrameworkResource,
	}
}

func (p *unifiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewNetworkDataSource,
		NewAccountDataSource,
		NewAPGroupDataSource,
		NewDNSRecordDataSource,
		NewPortProfileDataSource,
		NewRadiusProfileDataSource,
		NewUserGroupDataSource,
	}
}

func (p *unifiProvider) EphemeralResources(
	ctx context.Context,
) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}
