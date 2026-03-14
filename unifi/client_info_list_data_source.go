package unifi

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gounifi "github.com/ubiquiti-community/go-unifi/unifi"
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
			"clients": schema.ListNestedAttribute{
				MarkdownDescription: "List of active clients on the network.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: models.Attributes(),
				},
			},
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

	if client, ok := req.ProviderData.(*Client); ok {
		d.client = client
	} else {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
	}
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
		var notFoundErr *gounifi.NotFoundError
		if errors.As(err, &notFoundErr) {
			// The active clients endpoint is not available on all UniFi controller
			// versions or configurations. Return an empty list with a warning rather
			// than failing the plan entirely.
			resp.Diagnostics.AddWarning(
				"Active Clients Endpoint Not Available",
				"The active clients API endpoint returned 404. This may mean the feature "+
					"is not enabled or supported on your controller. An empty client list will be returned. "+
					"Original error: "+err.Error(),
			)
			clientInfoList = gounifi.ClientList{}
		} else {
			resp.Diagnostics.AddError(
				"Error Reading Clients",
				"Could not read active clients: "+err.Error(),
			)
			return
		}
	}

	clientObjects := make([]basetypes.ObjectValue, len(clientInfoList))
	for i, ci := range clientInfoList {
		v := models.ClientInfoAttrValues(ctx, &ci)
		o, d := types.ObjectValue(models.AttributeTypes(), v)
		if resp.Diagnostics.Append(d...); resp.Diagnostics.HasError() {
			return
		}
		clientObjects[i] = o
	}

	if clist, d := types.ListValueFrom(
		ctx,
		types.ObjectType{AttrTypes: models.AttributeTypes()},
		clientObjects,
	); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	} else {
		data.Clients = clist
	}

	data.Site = types.StringValue(site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
