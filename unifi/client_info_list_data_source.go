package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/models"
)

var _ datasource.DataSource = &clientInfoListDataSource{}

func NewClientInfoListDataSource() datasource.DataSource {
	return &clientInfoListDataSource{}
}

type clientInfoListDataSource struct {
	client *Client
}

type clientInfoListDataSourceModel struct {
	Site    types.String `tfsdk:"site"`
	Clients types.List   `tfsdk:"clients"`
}

func (d *clientInfoListDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_info_list"
}

func (d *clientInfoListDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all active clients on the network.",

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to retrieve clients from.",
				Optional:            true,
				Computed:            true,
			},
			"clients": models.ClientInfoListAttribute(),
		},
	}
}

func (d *clientInfoListDataSource) Configure(
	ctx context.Context,
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

func (d *clientInfoListDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data clientInfoListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	clientInfoList, err := d.client.ListClientInfo(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Clients",
			"Could not read active clients: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(models.ClientListValue(ctx, clientInfoList, &data.Clients)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Site = types.StringValue(site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stringValueOrNull returns a types.StringValue if the string is non-empty, otherwise types.StringNull.
func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
