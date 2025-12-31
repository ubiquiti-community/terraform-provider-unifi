package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &apGroupDataSource{}

func NewAPGroupDataSource() datasource.DataSource {
	return &apGroupDataSource{}
}

type apGroupDataSource struct {
	client *Client
}

type apGroupDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Site       types.String `tfsdk:"site"`
	Name       types.String `tfsdk:"name"`
	DeviceMacs types.List   `tfsdk:"device_macs"`
}

func (d *apGroupDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_ap_group"
}

func (d *apGroupDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for access point groups.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this AP group.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the AP group is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the AP group to look up.",
				Required:            true,
			},
			"device_macs": schema.ListAttribute{
				MarkdownDescription: "List of device MAC addresses in the AP group.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *apGroupDataSource) Configure(
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

func (d *apGroupDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data apGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	apGroups, err := d.client.ListAPGroup(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading AP Groups",
			"Could not read AP groups: "+err.Error(),
		)
		return
	}

	var apGroup *unifi.APGroup
	for _, group := range apGroups {
		if group.Name == name {
			apGroup = &group
			break
		}
	}

	if apGroup == nil {
		resp.Diagnostics.AddError(
			"AP Group Not Found",
			fmt.Sprintf("AP group with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(apGroup.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(apGroup.Name)
	deviceMacList := make([]types.String, len(apGroup.DeviceMacs))
	for i, v := range apGroup.DeviceMacs {
		deviceMacList[i] = types.StringValue(v)
	}
	deviceMacs, diags := types.ListValueFrom(ctx, types.StringType, deviceMacList)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	data.DeviceMacs = deviceMacs

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
