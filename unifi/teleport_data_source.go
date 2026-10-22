package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

var _ datasource.DataSource = &teleportDataSource{}

func NewTeleportDataSource() datasource.DataSource {
	return &teleportDataSource{}
}

type teleportDataSource struct {
	client *Client
}

type teleportDataSourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Site       types.String   `tfsdk:"site"`
	Enabled    types.Bool     `tfsdk:"enabled"`
	SubnetCidr types.String   `tfsdk:"subnet_cidr"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func (d *teleportDataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_teleport"
}

func (d *teleportDataSource) Schema(
	ctx context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the Teleport settings of a site.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the data source. It is the name of the site.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to read the Teleport settings from. " +
					"Defaults to the provider's configured site.",
				Optional: true,
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether Teleport is enabled.",
				Computed:            true,
			},
			"subnet_cidr": schema.StringAttribute{
				MarkdownDescription: "The subnet, in CIDR notation, used for Teleport. " +
					"Null when no subnet is set.",
				Computed: true,
			},
			"timeouts": timeouts.Attributes(ctx),
		},
	}
}

func (d *teleportDataSource) Configure(
	_ context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	d.client = client
}

func (d *teleportDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data teleportDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	_, teleport, err := unifi.GetSetting[*settings.Teleport](d.client.ApiClient, ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Teleport Settings",
			"Could not read the Teleport settings of site "+site+": "+err.Error(),
		)
		return
	}

	teleportToDataSourceModel(teleport, &data, site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// teleportToDataSourceModel copies the API setting into the model. An empty
// subnet becomes null, as the controller reports "" when none is set.
func teleportToDataSourceModel(
	teleport *settings.Teleport,
	data *teleportDataSourceModel,
	site string,
) {
	data.ID = types.StringValue(site)
	data.Site = types.StringValue(site)
	data.Enabled = types.BoolValue(teleport.Enabled)
	data.SubnetCidr = stringOrNull(teleport.SubnetCidr)
}
