package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &portProfileDataSource{}

func NewPortProfileDataSource() datasource.DataSource {
	return &portProfileDataSource{}
}

type portProfileDataSource struct {
	client *Client
}

type portProfileDataSourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Site                 types.String `tfsdk:"site"`
	Name                 types.String `tfsdk:"name"`
	Forward              types.String `tfsdk:"forward"`
	NativeNetworkconfID  types.String `tfsdk:"native_networkconf_id"`
	TaggedNetworkconfIDs types.Set    `tfsdk:"tagged_networkconf_ids"`
}

func (d *portProfileDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_port_profile"
}

func (d *portProfileDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for port profiles.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this port profile.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the port profile is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the port profile to look up.",
				Required:            true,
			},
		},
	}
}

func (d *portProfileDataSource) Configure(
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

func (d *portProfileDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data portProfileDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	portProfiles, err := d.client.ListPortProfile(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Port Profiles",
			"Could not read port profiles: "+err.Error(),
		)
		return
	}

	var portProfile *unifi.PortProfile
	for _, profile := range portProfiles {
		if profile.Name == name {
			portProfile = &profile
			break
		}
	}

	if portProfile == nil {
		resp.Diagnostics.AddError(
			"Port Profile Not Found",
			fmt.Sprintf("Port profile with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(portProfile.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(portProfile.Name)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
