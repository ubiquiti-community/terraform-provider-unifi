package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &userDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// userDataSource defines the data source implementation.
type userDataSource struct {
	client *Client
}

// userDataSourceModel describes the data source data model.
type userDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	MAC            types.String `tfsdk:"mac"`
	Name           types.String `tfsdk:"name"`
	UserGroupID    types.String `tfsdk:"user_group_id"`
	Note           types.String `tfsdk:"note"`
	FixedIP        types.String `tfsdk:"fixed_ip"`
	NetworkID      types.String `tfsdk:"network_id"`
	Blocked        types.Bool   `tfsdk:"blocked"`
	DevIDOverride  types.Int64  `tfsdk:"dev_id_override"`
	Hostname       types.String `tfsdk:"hostname"`
	IP             types.String `tfsdk:"ip"`
	LocalDNSRecord types.String `tfsdk:"local_dns_record"`
}

func (d *userDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Retrieves properties of a user (or "client" in the UI) of the network by MAC address.`,

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the user is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the user.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the user.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the user.",
				Computed:            true,
			},
			"user_group_id": schema.StringAttribute{
				MarkdownDescription: "The user group ID for the user.",
				Computed:            true,
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the user.",
				Computed:            true,
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "Fixed IPv4 address set for this user.",
				Computed:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this user.",
				Computed:            true,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this user should be blocked from the network.",
				Computed:            true,
			},
			"dev_id_override": schema.Int64Attribute{
				MarkdownDescription: "Override the device fingerprint.",
				Computed:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the user.",
				Computed:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The IP address of the user.",
				Computed:            true,
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "The local DNS record for this user.",
				Computed:            true,
			},
		},
	}
}

func (d *userDataSource) Configure(
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

func (d *userDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var config userDataSourceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	mac := config.MAC.ValueString()

	// Get user by MAC address first to get IP address
	macResp, err := d.client.GetUserByMAC(ctx, site, strings.ToLower(mac))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User by MAC",
			"Could not read user with MAC "+mac+": "+err.Error(),
		)
		return
	}

	// Get full user details by ID
	user, err := d.client.GetUser(ctx, site, macResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading User",
			"Could not read user with ID "+macResp.ID+": "+err.Error(),
		)
		return
	}

	// For some reason the IP address is only on the MAC endpoint
	user.IP = macResp.IP

	// Convert to model
	var state userDataSourceModel

	state.ID = types.StringValue(user.ID)
	state.Site = types.StringValue(site)
	state.MAC = types.StringValue(user.MAC)

	if user.Name != "" {
		state.Name = types.StringValue(user.Name)
	} else {
		state.Name = types.StringNull()
	}

	if user.UserGroupID != "" {
		state.UserGroupID = types.StringValue(user.UserGroupID)
	} else {
		state.UserGroupID = types.StringNull()
	}

	if user.Note != "" {
		state.Note = types.StringValue(user.Note)
	} else {
		state.Note = types.StringNull()
	}

	// Handle fixed IP
	if user.UseFixedIP && user.FixedIP != "" {
		state.FixedIP = types.StringValue(user.FixedIP)
	} else {
		state.FixedIP = types.StringNull()
	}

	if user.NetworkID != "" {
		state.NetworkID = types.StringValue(user.NetworkID)
	} else {
		state.NetworkID = types.StringNull()
	}

	state.Blocked = types.BoolValue(user.Blocked)

	if user.DevIdOverride != 0 {
		state.DevIDOverride = types.Int64Value(int64(user.DevIdOverride))
	} else {
		state.DevIDOverride = types.Int64Null()
	}

	if user.Hostname != "" {
		state.Hostname = types.StringValue(user.Hostname)
	} else {
		state.Hostname = types.StringNull()
	}

	if user.IP != "" {
		state.IP = types.StringValue(user.IP)
	} else {
		state.IP = types.StringNull()
	}

	// Handle local DNS record
	if user.LocalDNSRecordEnabled && user.LocalDNSRecord != "" {
		state.LocalDNSRecord = types.StringValue(user.LocalDNSRecord)
	} else {
		state.LocalDNSRecord = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
