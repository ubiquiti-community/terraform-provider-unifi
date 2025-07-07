package unifi

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

const (
	ApiKeyDescription = "API key for the Unifi controller. Can be specified with the `UNIFI_API_KEY` " +
		"environment variable. If this is set, the `username` and `password` fields are ignored."
	UserNameDescription = "Local user name for the Unifi controller API. Can be specified with the `UNIFI_USERNAME` " +
		"environment variable."
	PasswordDescription = "Password for the user accessing the API. Can be specified with the `UNIFI_PASSWORD` " +
		"environment variable."
	ApiUrlDescription = "URL of the controller API. Can be specified with the `UNIFI_API` environment variable. " +
		"You should **NOT** supply the path (`/api`), the SDK will discover the appropriate paths. This is " +
		"to support UDM Pro style API paths as well as more standard controller paths."
	SiteDescription = "The site in the Unifi controller this provider will manage. Can be specified with " +
		"the `UNIFI_SITE` environment variable. Default: `default`"
	AllowInsecureDescription = "Skip verification of TLS certificates of API requests. You may need to set this to `true` " +
		"if you are using your local API without setting up a signed certificate. Can be specified with the " +
		"`UNIFI_INSECURE` environment variable."
)

// frameworkProvider is a type that implements the terraform-plugin-framework
// provider.Provider interface. Someday, this will probably encompass the entire
// behavior of the unifi provider. Today, it is a small but growing subset.
type frameworkProvider struct {
	defaultSiteName *string
}

var (
	_ provider.Provider                       = &frameworkProvider{}
	_ provider.ProviderWithEphemeralResources = &frameworkProvider{}
)

// FrameworkProviderConfig is a helper type for extracting the provider
// configuration from the provider block.
type FrameworkProviderConfig struct {
	ApiKey        types.String `tfsdk:"api_key"`
	User          types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	ApiUrl        types.String `tfsdk:"api_url"`
	Site          types.String `tfsdk:"site"`
	AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
}

// NewFrameworkProvider is a helper function for initializing the portion of
// the unifi provider implemented via the terraform-plugin-framework.
func NewFrameworkProvider() provider.Provider {
	return &frameworkProvider{}
}

// NewFrameworkProviderWithDefaultOrg is a helper function for
// initializing a framework provider with a default site name.
func NewFrameworkProviderWithDefaultSite(defaultSiteName string) provider.Provider {
	return &frameworkProvider{defaultSiteName: &defaultSiteName}
}

// Metadata (a Provider interface function) lets the provider identify itself.
// Resources and data sources can access this information from their request
// objects.
func (p *frameworkProvider) Metadata(
	_ context.Context,
	_ provider.MetadataRequest,
	res *provider.MetadataResponse,
) {
	res.TypeName = "unifi"
}

// Schema (a Provider interface function) returns the schema for the Terraform
// block that configures the provider itself.
func (p *frameworkProvider) Schema(
	_ context.Context,
	_ provider.SchemaRequest,
	res *provider.SchemaResponse,
) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: ApiKeyDescription,
				Optional:            true,
				// Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: UserNameDescription,
				Optional:            true,
				// Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: PasswordDescription,
				Optional:            true,
				// Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				MarkdownDescription: ApiUrlDescription,
				Required:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: SiteDescription,
				Optional:            true,
			},
			"allow_insecure": schema.BoolAttribute{
				MarkdownDescription: AllowInsecureDescription,
				Optional:            true,
			},
		},
	}
}

// Configure (a Provider interface function) sets up the HCP Terraform client per the
// specified provider configuration block and env vars.
func (p *frameworkProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	res *provider.ConfigureResponse,
) {
	var data FrameworkProviderConfig
	diags := req.Config.Get(ctx, &data)

	res.Diagnostics.Append(diags...)
	if res.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Configuring Synology provider")

	res.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if res.Diagnostics.HasError() {
		return
	}
	apiUrl := data.ApiUrl.ValueString()
	if apiUrl == "" {
		if v := os.Getenv("UNIFI_API"); v != "" {
			apiUrl = v
		}
	}
	user := data.User.ValueString()
	if user == "" {
		if v := os.Getenv("UNIFI_USERNAME"); v != "" {
			user = v
		}
	}
	pass := data.Password.ValueString()
	if pass == "" {
		if v := os.Getenv("UNIFI_PASSWORD"); v != "" {
			pass = v
		}
	}
	apikey := data.ApiKey.ValueString()
	if apikey == "" {
		if v := os.Getenv("UNIFI_API_KEY"); v != "" {
			apikey = v
		}
	}
	insecure := data.AllowInsecure.ValueBool()
	if !insecure {
		if v := os.Getenv("UNIFI_INSECURE"); v != "" {
			insecure = v == "true"
		}
	}
	site := data.Site.ValueString()
	if site == "" {
		if v := os.Getenv("UNIFI_SITE"); v != "" {
			site = v
		}
	}
	if site == "" {
		if p.defaultSiteName != nil {
			site = *p.defaultSiteName
		} else {
			site = "default"
		}
	}

	// Create the go-unifi client directly
	unifiClient := &unifi.Client{}
	err := unifiClient.SetBaseURL(apiUrl)
	if err != nil {
		res.Diagnostics.AddError(
			"Invalid API URL",
			"Could not set base URL: "+err.Error(),
		)
		return
	}

	httpClient := &http.Client{}
	if insecure {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = transport
	}
	unifiClient.SetHTTPClient(httpClient)

	// Set authentication
	if apikey != "" {
		unifiClient.SetAPIKey(apikey)
	}
	// TODO: Add login with username/password if needed
	// For now, assume API key authentication

	configuredClient := &Client{
		Client: unifiClient,
		Site:   site,
	}

	res.DataSourceData = configuredClient
	res.ResourceData = configuredClient
	res.EphemeralResourceData = configuredClient
}

func (p *frameworkProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserFrameworkDataSource,
		NewNetworkFrameworkDataSource,
		NewAccountDataSource,
		NewAPGroupDataSource,
		NewDNSRecordDataSource,
		NewPortProfileDataSource,
		NewRadiusProfileDataSource,
		NewUserGroupDataSource,
	}
}

func (p *frameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewNetworkResource,
		NewWLANFrameworkResource,
		NewUserFrameworkResource,
		NewSiteFrameworkResource,
		NewUserGroupFrameworkResource,
		NewDNSRecordFrameworkResource,
		NewAccountFrameworkResource,
		NewStaticRouteFrameworkResource,
		NewDynamicDNSResource,
		NewFirewallGroupFrameworkResource,
		NewPortProfileFrameworkResource,
		NewDeviceFrameworkResource,
		NewFirewallRuleResource,
		NewPortForwardResource,
		NewRadiusProfileResource,
		NewSettingMgmtResource,
		NewSettingRadiusResource,
		NewSettingUSGResource,
	}
}

func (p *frameworkProvider) EphemeralResources(
	ctx context.Context,
) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}
