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
	ID             types.String `tfsdk:"id"`
	Site           types.String `tfsdk:"site"`
	MAC            types.String `tfsdk:"mac"`
	Name           types.String `tfsdk:"name"`
	DisplayName    types.String `tfsdk:"display_name"`
	QOSRate        types.Object `tfsdk:"qos_rate"`
	Note           types.String `tfsdk:"note"`
	FixedIP        types.String `tfsdk:"fixed_ip"`
	FixedApMAC     types.String `tfsdk:"fixed_ap_mac"`
	NetworkID      types.String `tfsdk:"network_id"`
	Groups         types.List   `tfsdk:"groups"`
	Blocked        types.Bool   `tfsdk:"blocked"`
	LocalDNSRecord types.String `tfsdk:"local_dns_record"`
	Hostname       types.String `tfsdk:"hostname"`
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
			"qos_rate": schema.SingleNestedAttribute{
				MarkdownDescription: "QoS rate limiting configuration from the client's group.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "The ID of the client group.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the client group.",
						Computed:            true,
					},
					"max_up": schema.Int64Attribute{
						MarkdownDescription: "Maximum upload rate in kbps.",
						Computed:            true,
					},
					"max_down": schema.Int64Attribute{
						MarkdownDescription: "Maximum download rate in kbps.",
						Computed:            true,
					},
				},
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
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of network members group names for this client.",
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
		group, err := d.client.GetClientGroup(ctx, site, client.UserGroupID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Client Group",
				fmt.Sprintf("Could not read client group %q: %s", client.UserGroupID, err.Error()),
			)
			return
		}
		qos := qosRateModel{
			ID:      types.StringValue(group.ID),
			Name:    types.StringValue(group.Name),
			MaxUp:   types.Int64PointerValue(group.QOSRateMaxUp),
			MaxDown: types.Int64PointerValue(group.QOSRateMaxDown),
		}
		var objDiags diag.Diagnostics
		state.QOSRate, objDiags = types.ObjectValueFrom(ctx, qosRateModel{}.AttributeTypes(), qos)
		resp.Diagnostics.Append(objDiags...)
	} else {
		state.QOSRate = types.ObjectNull(qosRateModel{}.AttributeTypes())
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

	// Resolve NetworkMembersGroupIDs to tag names
	if len(client.NetworkMembersGroupIDs) > 0 {
		groups, err := d.client.ListNetworkMembersGroups(ctx, site)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Listing Groups",
				"Could not list network members groups: "+err.Error(),
			)
			return
		}
		idToName := make(map[string]string, len(groups))
		for _, g := range groups {
			idToName[g.ID] = g.Name
		}
		elements := make([]attr.Value, 0, len(client.NetworkMembersGroupIDs))
		for _, id := range client.NetworkMembersGroupIDs {
			if name, ok := idToName[id]; ok {
				elements = append(elements, types.StringValue(name))
			}
		}
		if len(elements) > 0 {
			var listDiags diag.Diagnostics
			state.Groups, listDiags = types.ListValue(types.StringType, elements)
			resp.Diagnostics.Append(listDiags...)
		} else {
			state.Groups = types.ListNull(types.StringType)
		}
	} else {
		state.Groups = types.ListNull(types.StringType)
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
