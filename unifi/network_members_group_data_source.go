package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	gounifi "github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &networkMembersGroupListDataSource{}

func NewNetworkMembersGroupListDataSource() datasource.DataSource {
	return &networkMembersGroupListDataSource{}
}

type networkMembersGroupListDataSource struct {
	client *Client
}

type networkMembersGroupListDataSourceModel struct {
	Site   types.String `tfsdk:"site"`
	Groups types.List   `tfsdk:"groups"`
}

func (d *networkMembersGroupListDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_network_members_group_list"
}

func networkMembersGroupAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":      types.StringType,
		"name":    types.StringType,
		"members": types.ListType{ElemType: types.StringType},
		"type":    types.StringType,
	}
}

func networkMembersGroupSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the network members group.",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the network members group.",
			Computed:            true,
		},
		"members": schema.ListAttribute{
			MarkdownDescription: "The list of member identifiers in the group.",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"type": schema.StringAttribute{
			MarkdownDescription: "The type of the network members group.",
			Computed:            true,
		},
	}
}

func (d *networkMembersGroupListDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of all network members groups.",

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to retrieve network members groups from.",
				Optional:            true,
				Computed:            true,
			},
			"groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of network members groups.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: networkMembersGroupSchemaAttributes(),
				},
			},
		},
	}
}

func (d *networkMembersGroupListDataSource) Configure(
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

func networkMembersGroupAttrValues(group *gounifi.NetworkMembersGroup) map[string]attr.Value {
	memberValues := make([]attr.Value, len(group.Members))
	for i, m := range group.Members {
		memberValues[i] = types.StringValue(m)
	}

	var membersList basetypes.ListValue
	if len(group.Members) > 0 {
		membersList = types.ListValueMust(types.StringType, memberValues)
	} else {
		membersList = types.ListValueMust(types.StringType, []attr.Value{})
	}

	return map[string]attr.Value{
		"id":      types.StringValue(group.ID),
		"name":    types.StringValue(group.Name),
		"members": membersList,
		"type":    types.StringValue(group.Type),
	}
}

func (d *networkMembersGroupListDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data networkMembersGroupListDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	groups, err := d.client.ListNetworkMembersGroups(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Members Groups",
			"Could not read network members groups: "+err.Error(),
		)
		return
	}

	groupObjects := make([]basetypes.ObjectValue, len(groups))
	for i, g := range groups {
		v := networkMembersGroupAttrValues(&g)
		o, d := types.ObjectValue(networkMembersGroupAttrTypes(), v)
		if resp.Diagnostics.Append(d...); resp.Diagnostics.HasError() {
			return
		}
		groupObjects[i] = o
	}

	if glist, d := types.ListValueFrom(
		ctx,
		types.ObjectType{AttrTypes: networkMembersGroupAttrTypes()},
		groupObjects,
	); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	} else {
		data.Groups = glist
	}

	data.Site = types.StringValue(site)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
