package unifi

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
)

type unifiProvider struct {
	version string
}

type unifiProviderModel struct {
	APIKey        types.String `tfsdk:"api_key"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	APIUrl        types.String `tfsdk:"api_url"`
	Site          types.String `tfsdk:"site"`
	AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
}

func New() provider.Provider {
	return &unifiProvider{}
}

func (p *unifiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The UniFi provider is used to configure a UniFi controller.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "API key for the Unifi controller. Can be specified with the `UNIFI_API_KEY` " +
					"environment variable. If this is set, the `username` and `password` fields are ignored.",
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(32),
				},
			},
			"username": schema.StringAttribute{
				Description: "Local user name for the Unifi controller API. Can be specified with the `UNIFI_USERNAME` " +
					"environment variable.",
				Optional: true,
			},
			"password": schema.StringAttribute{
				Description: "Password for the user accessing the API. Can be specified with the `UNIFI_PASSWORD` " +
					"environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"api_url": schema.StringAttribute{
				Description: "URL of the controller API. Can be specified with the `UNIFI_API` environment variable. " +
					"You should **NOT** supply the path (`/api`), the SDK will discover the appropriate paths. This is " +
					"to support UDM Pro style API paths as well as more standard controller paths.",
				Optional: true,
			},
			"site": schema.StringAttribute{
				Description: "The site in the Unifi controller this provider will manage. Can be specified with " +
					"the `UNIFI_SITE` environment variable. Default: `default`",
				Optional: true,
			},
			"allow_insecure": schema.BoolAttribute{
				Description: "Skip verification of TLS certificates of API requests. You may need to set this to `true` " +
					"if you are using your local API without setting up a signed certificate. Can be specified with the " +
					"`UNIFI_INSECURE` environment variable.",
				Optional: true,
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

	// Set defaults from environment variables
	apiKey := config.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("UNIFI_API_KEY")
	}

	username := config.Username.ValueString()
	if username == "" {
		username = os.Getenv("UNIFI_USERNAME")
	}

	password := config.Password.ValueString()
	if password == "" {
		password = os.Getenv("UNIFI_PASSWORD")
	}

	apiURL := config.APIUrl.ValueString()
	if apiURL == "" {
		apiURL = os.Getenv("UNIFI_API")
	}

	site := config.Site.ValueString()
	if site == "" {
		site = os.Getenv("UNIFI_SITE")
		if site == "" {
			site = "default"
		}
	}

	allowInsecure := config.AllowInsecure.ValueBool()
	if !config.AllowInsecure.IsNull() && !config.AllowInsecure.IsUnknown() {
		allowInsecure = config.AllowInsecure.ValueBool()
	} else {
		allowInsecure = os.Getenv("UNIFI_INSECURE") == "true"
	}

	// Validate required fields
	if apiURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing UniFi API URL",
			"The provider cannot create the UniFi API client as there is a missing or empty value for the UniFi API URL. "+
				"Set the api_url value in the configuration or use the UNIFI_API environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apiKey == "" {
		if username == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing UniFi API Username",
				"The provider cannot create the UniFi API client as there is a missing or empty value for the UniFi API username. "+
					"Set the username value in the configuration or use the UNIFI_USERNAME environment variable. "+
					"If either is already set, ensure the value is not empty.",
			)
		}

		if password == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing UniFi API Password",
				"The provider cannot create the UniFi API client as there is a missing or empty value for the UniFi API password. "+
					"Set the password value in the configuration or use the UNIFI_PASSWORD environment variable. "+
					"If either is already set, ensure the value is not empty.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create UniFi client using the lazy client pattern from the existing provider
	// We'll create a client wrapper that implements the same interface
	client := &frameworkClient{
		apiKey:        apiKey,
		username:      username,
		password:      password,
		baseURL:       apiURL,
		site:          site,
		allowInsecure: allowInsecure,
	}

	// Make the UniFi client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *unifiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Add new framework resources here as they are implemented
		NewWlanResource,
	}
}

func (p *unifiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Add new framework data sources here as they are implemented
	}
}

func (p *unifiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "unifi"
	resp.Version = p.version
}

// frameworkClient is a simple client wrapper that will be used by Framework resources
// This can be extended later to implement the same interface as the SDK v2 client
type frameworkClient struct {
	apiKey        string
	username      string
	password      string
	baseURL       string
	site          string
	allowInsecure bool
}