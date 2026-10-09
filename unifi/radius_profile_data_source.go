package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/datasource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var _ datasource.DataSource = &radiusProfileDataSource{}

func NewRadiusProfileDataSource() datasource.DataSource {
	return &radiusProfileDataSource{}
}

type radiusProfileDataSource struct {
	client *Client
}

// radiusProfileDataSourceModel describes the data source data model. The
// `interim_update` and `vlan` objects share their sub-models and attribute
// types with the resource (see radius_profile_resource.go).
type radiusProfileDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Site              types.String `tfsdk:"site"`
	Name              types.String `tfsdk:"name"`
	AccountingEnabled types.Bool   `tfsdk:"accounting_enabled"`
	InterimUpdate     types.Object `tfsdk:"interim_update"`
	UseUSGAcctServer  types.Bool   `tfsdk:"use_usg_acct_server"`
	UseUSGAuthServer  types.Bool   `tfsdk:"use_usg_auth_server"`
	Vlan              types.Object `tfsdk:"vlan"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
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
			"interim_update": schema.SingleNestedAttribute{
				MarkdownDescription: "RADIUS interim accounting update settings.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether interim updates are enabled.",
						Computed:            true,
					},
					"interval": schema.StringAttribute{
						MarkdownDescription: "The interim update interval, as a Go duration string.",
						CustomType:          timetypes.GoDurationType{},
						Computed:            true,
					},
				},
			},
			"use_usg_acct_server": schema.BoolAttribute{
				MarkdownDescription: "Whether to use USG as accounting server.",
				Computed:            true,
			},
			"use_usg_auth_server": schema.BoolAttribute{
				MarkdownDescription: "Whether to use USG as authentication server.",
				Computed:            true,
			},
			"vlan": schema.SingleNestedAttribute{
				MarkdownDescription: "Dynamic VLAN assignment settings.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Whether VLAN is enabled.",
						Computed:            true,
					},
					"wlan_mode": schema.StringAttribute{
						MarkdownDescription: "The VLAN WLAN mode.",
						Computed:            true,
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx),
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

	name := data.Name.ValueString()

	// Get RADIUS profiles from API
	radiusProfiles, err := d.client.ListRADIUSProfile(ctx, site)
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
	data.UseUSGAcctServer = types.BoolValue(radiusProfile.UseUsgAcctServer)
	data.UseUSGAuthServer = types.BoolValue(radiusProfile.UseUsgAuthServer)

	interimUpdate, diags := types.ObjectValueFrom(
		ctx,
		radiusProfileInterimUpdateAttrTypes(),
		radiusProfileInterimUpdateModel{
			Enabled:  types.BoolValue(radiusProfile.InterimUpdateEnabled),
			Interval: util.DurationPtrValue(radiusProfile.InterimUpdateInterval, time.Second),
		},
	)
	resp.Diagnostics.Append(diags...)
	data.InterimUpdate = interimUpdate

	vlan, diags := types.ObjectValueFrom(ctx, radiusProfileVlanAttrTypes(), radiusProfileVlanModel{
		Enabled:  types.BoolValue(radiusProfile.VLANEnabled),
		WlanMode: types.StringValue(radiusProfile.VLANWLANMode),
	})
	resp.Diagnostics.Append(diags...)
	data.Vlan = vlan

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
