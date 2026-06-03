package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var _ datasource.DataSource = &firewallZoneDataSource{}

func NewFirewallZoneDataSource() datasource.DataSource {
	return &firewallZoneDataSource{}
}

type firewallZoneDataSource struct {
	client *Client
}

type firewallZoneDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Site       types.String `tfsdk:"site"`
	Name       types.String `tfsdk:"name"`
	ZoneKey    types.String `tfsdk:"zone_key"`
	NetworkIDs types.List   `tfsdk:"network_ids"`
}

func (d *firewallZoneDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_firewall_zone"
}

func (d *firewallZoneDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Data source for UniFi firewall zones (zone-based firewall, UniFi Network 8.x+). " +
			"Use this to look up zone IDs by name for use in `unifi_firewall_policy` resources.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the firewall zone.",
				Computed:            true,
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the UniFi site.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The display name of the firewall zone (e.g. `Internal`, `External`).",
				Required:            true,
			},
			"zone_key": schema.StringAttribute{
				MarkdownDescription: "The internal key of the zone (e.g. `lan`, `wan`).",
				Computed:            true,
			},
			"network_ids": schema.ListAttribute{
				MarkdownDescription: "List of network IDs assigned to this zone.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *firewallZoneDataSource) Configure(
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

func (d *firewallZoneDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data firewallZoneDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = d.client.Site
	}

	name := data.Name.ValueString()

	zones, err := d.client.ListFirewallZone(ctx, site)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Firewall Zones",
			"Could not list firewall zones: "+err.Error(),
		)
		return
	}

	var zone *unifi.FirewallZone
	for _, z := range zones {
		if z.Name == name {
			zone = &z
			break
		}
	}

	if zone == nil {
		resp.Diagnostics.AddError(
			"Firewall Zone Not Found",
			fmt.Sprintf("No firewall zone with name %q found on site %q.", name, site),
		)
		return
	}

	data.ID = types.StringValue(zone.ID)
	data.Site = types.StringValue(site)
	data.Name = types.StringValue(zone.Name)
	data.ZoneKey = types.StringValue(zone.ZoneKey)

	networkIDs, diags := types.ListValueFrom(ctx, types.StringType, zone.NetworkIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.NetworkIDs = networkIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
