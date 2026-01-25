package unifi

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &clientDataSource{}

func NewClientDataSource() datasource.DataSource {
	return &clientDataSource{}
}

// clientDataSource defines the data source implementation.
type clientDataSource struct {
	client *Client
}

// clientDataSourceModel describes the data source data model.
type clientDataSourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Site                   types.String `tfsdk:"site"`
	MAC                    types.String `tfsdk:"mac"`
	Name                   types.String `tfsdk:"name"`
	DisplayName            types.String `tfsdk:"display_name"`
	GroupID                types.String `tfsdk:"group_id"`
	Note                   types.String `tfsdk:"note"`
	FixedIP                types.String `tfsdk:"fixed_ip"`
	FixedApMAC             types.String `tfsdk:"fixed_ap_mac"`
	NetworkID              types.String `tfsdk:"network_id"`
	NetworkMembersGroupIDs types.List   `tfsdk:"network_members_group_ids"`
	Blocked                types.Bool   `tfsdk:"blocked"`
	LocalDNSRecord         types.String `tfsdk:"local_dns_record"`
	Hostname               types.String `tfsdk:"hostname"`
}

func (d *clientDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

func (d *clientDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Retrieves properties of a client of the network by MAC address.`,

		Attributes: map[string]schema.Attribute{
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site the client is associated with.",
				Optional:            true,
				Computed:            true,
			},
			"mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the client.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the client.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the client.",
				Computed:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the client.",
				Computed:            true,
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The group ID to attach to the client (controls QoS and other group-based settings).",
				Computed:            true,
			},
			"note": schema.StringAttribute{
				MarkdownDescription: "A note with additional information for the client.",
				Computed:            true,
			},
			"fixed_ip": schema.StringAttribute{
				MarkdownDescription: "A fixed IPv4 address for this client.",
				Computed:            true,
			},
			"fixed_ap_mac": schema.StringAttribute{
				MarkdownDescription: "The MAC address of the access point to which this client should be fixed.",
				Computed:            true,
			},
			"network_id": schema.StringAttribute{
				MarkdownDescription: "The network ID for this client.",
				Computed:            true,
			},
			"network_members_group_ids": schema.ListAttribute{
				MarkdownDescription: "List of network member group IDs for this client.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"blocked": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether this client should be blocked from the network.",
				Computed:            true,
			},
			"local_dns_record": schema.StringAttribute{
				MarkdownDescription: "Specifies the local DNS record for this client.",
				Computed:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "The hostname of the client.",
				Computed:            true,
			},
		},
	}
}

func (d *clientDataSource) Configure(
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

func (d *clientDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var config clientDataSourceModel

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

	// Get client by MAC address first to get IP address
	macResp, err := d.client.GetClientByMAC(ctx, site, strings.ToLower(mac))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client by MAC",
			"Could not read client with MAC "+mac+": "+err.Error(),
		)
		return
	}

	// Get full client details by ID
	client, err := d.client.GetClient(ctx, site, macResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Client",
			"Could not read client with ID "+macResp.ID+": "+err.Error(),
		)
		return
	}

	// For some reason the IP address is only on the MAC endpoint
	client.LastSeen = macResp.LastSeen

	// Convert to model
	var state clientDataSourceModel

	state.ID = types.StringValue(client.ID)
	state.Site = types.StringValue(site)
	state.MAC = types.StringValue(client.MAC)

	if client.Name != "" {
		state.Name = types.StringValue(client.Name)
	} else {
		state.Name = types.StringNull()
	}

	if client.DisplayName != "" {
		state.DisplayName = types.StringValue(client.DisplayName)
	} else {
		state.DisplayName = types.StringNull()
	}

	if client.UserGroupID != "" {
		state.GroupID = types.StringValue(client.UserGroupID)
	} else {
		state.GroupID = types.StringNull()
	}

	if client.Note != "" {
		state.Note = types.StringValue(client.Note)
	} else {
		state.Note = types.StringNull()
	}

	if client.FixedIP != "" {
		state.FixedIP = types.StringValue(client.FixedIP)
	} else {
		state.FixedIP = types.StringNull()
	}

	if client.FixedApMAC != "" {
		state.FixedApMAC = types.StringValue(client.FixedApMAC)
	} else {
		state.FixedApMAC = types.StringNull()
	}

	if client.VirtualNetworkOverrideID != "" {
		state.NetworkID = types.StringValue(client.VirtualNetworkOverrideID)
	} else if client.NetworkID != "" {
		state.NetworkID = types.StringValue(client.NetworkID)
	} else {
		state.NetworkID = types.StringNull()
	}

	// Convert NetworkMembersGroupIDs from []string to List
	if len(client.NetworkMembersGroupIDs) > 0 {
		elements := make([]attr.Value, len(client.NetworkMembersGroupIDs))
		for i, id := range client.NetworkMembersGroupIDs {
			elements[i] = types.StringValue(id)
		}
		var listDiags diag.Diagnostics
		state.NetworkMembersGroupIDs, listDiags = types.ListValue(types.StringType, elements)
		resp.Diagnostics.Append(listDiags...)
	} else {
		state.NetworkMembersGroupIDs = types.ListNull(types.StringType)
	}

	state.Blocked = types.BoolPointerValue(client.Blocked)

	if client.LocalDNSRecord != "" {
		state.LocalDNSRecord = types.StringValue(client.LocalDNSRecord)
	} else {
		state.LocalDNSRecord = types.StringNull()
	}

	if client.Hostname != "" {
		state.Hostname = types.StringValue(client.Hostname)
	} else {
		state.Hostname = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
