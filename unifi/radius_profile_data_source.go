package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &radiusProfileDataSource{}

func NewRadiusProfileDataSource() datasource.DataSource {
	return &radiusProfileDataSource{}
}

type radiusProfileDataSource struct {
	client *Client
}

type radiusProfileDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Site                  types.String `tfsdk:"site"`
	Name                  types.String `tfsdk:"name"`
	AccountingEnabled     types.Bool   `tfsdk:"accounting_enabled"`
	InterimUpdateEnabled  types.Bool   `tfsdk:"interim_update_enabled"`
	InterimUpdateInterval types.Int64  `tfsdk:"interim_update_interval"`
	UseUSGAcctServer      types.Bool   `tfsdk:"use_usg_acct_server"`
	UseUSGAuthServer      types.Bool   `tfsdk:"use_usg_auth_server"`
	VlanEnabled           types.Bool   `tfsdk:"vlan_enabled"`
	VlanWlanMode          types.String `tfsdk:"vlan_wlan_mode"`
}

func (d *radiusProfileDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_radius_profile"
}

func (d *radiusProfileDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for RADIUS profiles.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this RADIUS profile.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the RADIUS profile is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the RADIUS profile to look up.",
				Required:            true,
			},
			"accounting_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether RADIUS accounting is enabled.",
				Computed:            true,
			},
			"interim_update_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether interim updates are enabled.",
				Computed:            true,
			},
			"interim_update_interval": schema.Int64Attribute{
				MarkdownDescription: "The interim update interval.",
				Computed:            true,
			},
			"use_usg_acct_server": schema.BoolAttribute{
				MarkdownDescription: "Whether to use USG as accounting server.",
				Computed:            true,
			},
			"use_usg_auth_server": schema.BoolAttribute{
				MarkdownDescription: "Whether to use USG as authentication server.",
				Computed:            true,
			},
			"vlan_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether VLAN is enabled.",
				Computed:            true,
			},
			"vlan_wlan_mode": schema.StringAttribute{
				MarkdownDescription: "The VLAN WLAN mode.",
				Computed:            true,
			},
		},
	}
}

func (d *radiusProfileDataSource) Configure(
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

func (d *radiusProfileDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data radiusProfileDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	// Get RADIUS profiles from API
	radiusProfiles, err := d.client.Client.ListRADIUSProfile(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading RADIUS Profiles",
			"Could not read RADIUS profiles: "+err.Error(),
		)
		return
	}

	var radiusProfile *unifi.RADIUSProfile
	for _, profile := range radiusProfiles {
		if profile.Name == name {
			radiusProfile = &profile
			break
		}
	}

	if radiusProfile == nil {
		resp.Diagnostics.AddError(
			"RADIUS Profile Not Found",
			fmt.Sprintf("RADIUS profile with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(radiusProfile.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(radiusProfile.Name)
	data.AccountingEnabled = types.BoolValue(radiusProfile.AccountingEnabled)
	data.InterimUpdateEnabled = types.BoolValue(radiusProfile.InterimUpdateEnabled)
	data.InterimUpdateInterval = types.Int64Value(int64(radiusProfile.InterimUpdateInterval))
	data.UseUSGAcctServer = types.BoolValue(radiusProfile.UseUsgAcctServer)
	data.UseUSGAuthServer = types.BoolValue(radiusProfile.UseUsgAuthServer)
	data.VlanEnabled = types.BoolValue(radiusProfile.VLANEnabled)
	data.VlanWlanMode = types.StringValue(radiusProfile.VLANWLANMode)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
