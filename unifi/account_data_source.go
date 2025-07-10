package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &accountDataSource{}

func NewAccountDataSource() datasource.DataSource {
	return &accountDataSource{}
}

type accountDataSource struct {
	client *Client
}

type accountDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Site             types.String `tfsdk:"site"`
	Name             types.String `tfsdk:"name"`
	Password         types.String `tfsdk:"password"`
	TunnelType       types.Int64  `tfsdk:"tunnel_type"`
	TunnelMediumType types.Int64  `tfsdk:"tunnel_medium_type"`
	NetworkID        types.String `tfsdk:"network_id"`
}

func (d *accountDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_account"
}

func (d *accountDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for RADIUS user accounts.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this account.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the account is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the account to look up.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password of the account.",
				Computed:            true,
				Sensitive:           true,
			},
			"tunnel_type": schema.Int64Attribute{
				MarkdownDescription: "See RFC2868 section 3.1.",
				Computed:            true,
			},
			"tunnel_medium_type": schema.Int64Attribute{
				MarkdownDescription: "See RFC2868 section 3.2.",
				Computed:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "ID of the network for this account.",
				Computed:            true,
			},
		},
	}
}

func (d *accountDataSource) Configure(
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

func (d *accountDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data accountDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	accounts, err := d.client.ListAccount(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Accounts",
			"Could not read accounts: "+err.Error(),
		)
		return
	}

	var account *unifi.Account
	for _, a := range accounts {
		if a.Name == name {
			account = &a
			break
		}
	}

	if account == nil {
		resp.Diagnostics.AddError(
			"Account Not Found",
			fmt.Sprintf("Account with name %s not found", name),
		)
		return
	}

	data.ID = types.StringValue(account.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(account.Name)
	data.Password = types.StringValue(account.XPassword)
	data.TunnelType = types.Int64Value(int64(account.TunnelType))
	data.TunnelMediumType = types.Int64Value(int64(account.TunnelMediumType))
	data.NetworkID = types.StringValue(account.NetworkID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
