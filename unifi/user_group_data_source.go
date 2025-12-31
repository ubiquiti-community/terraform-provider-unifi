package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &userGroupDataSource{}

func NewUserGroupDataSource() datasource.DataSource {
	return &userGroupDataSource{}
}

type userGroupDataSource struct {
	client *Client
}

type userGroupDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	Name           types.String `tfsdk:"name"`
	QOSRateMaxDown types.Int64  `tfsdk:"qos_rate_max_down"`
	QOSRateMaxUp   types.Int64  `tfsdk:"qos_rate_max_up"`
}

func (d *userGroupDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_user_group"
}

func (d *userGroupDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for user groups.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this user group.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the user group is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user group to look up.",
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

func (d *userGroupDataSource) Configure(
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

func (d *userGroupDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data userGroupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	userGroups, err := d.client.ListUserGroup(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User Groups",
			"Could not read user groups: "+err.Error(),
		)
		return
	}

	var userGroup *unifi.UserGroup
	for _, group := range userGroups {
		if group.Name == name {
			userGroup = &group
			break
		}
	}

	if userGroup == nil {
		resp.Diagnostics.AddError(
			"User Group Not Found",
			fmt.Sprintf("User group with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(userGroup.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(userGroup.Name)
	data.QOSRateMaxDown = types.Int64Value(int64(userGroup.QOSRateMaxDown))
	data.QOSRateMaxUp = types.Int64Value(int64(userGroup.QOSRateMaxUp))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
