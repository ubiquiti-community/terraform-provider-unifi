package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &clientQosRateDataSource{}

func NewClientQosRateDataSource() datasource.DataSource {
	return &clientQosRateDataSource{}
}

type clientQosRateDataSource struct {
	client *Client
}

type clientQosRateDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Name           types.String `tfsdk:"name"`
	QOSRateMaxDown types.Int64  `tfsdk:"qos_rate_max_down"`
	QOSRateMaxUp   types.Int64  `tfsdk:"qos_rate_max_up"`
}

func (d *clientQosRateDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client_qos_rate"
}

func (d *clientQosRateDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for client QOS rates.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this client QOS rate.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the client QOS rate is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client QOS rate to look up.",
				Required:            true,
			},
			"qos_rate_max_down": schema.Int64Attribute{
				MarkdownDescription: "The maximum download rate.",
				Computed:            true,
			},
			"qos_rate_max_up": schema.Int64Attribute{
				MarkdownDescription: "The maximum upload rate.",
				Computed:            true,
			},
		},
	}
}

func (d *clientQosRateDataSource) Configure(
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

func (d *clientQosRateDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data clientQosRateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	clientGroups, err := d.client.ListClientGroup(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client QOS Rates",
			"Could not read client QOS rates: "+err.Error(),
		)
		return
	}

	var clientGroup *unifi.ClientGroup
	for _, group := range clientGroups {
		if group.Name == name {
			clientGroup = &group
			break
		}
	}

	if clientGroup == nil {
		resp.Diagnostics.AddError(
			"Client QOS Rate Not Found",
			fmt.Sprintf("Client group with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(clientGroup.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(clientGroup.Name)
	data.QOSRateMaxDown = types.Int64PointerValue(clientGroup.QOSRateMaxDown)
	data.QOSRateMaxUp = types.Int64PointerValue(clientGroup.QOSRateMaxUp)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
