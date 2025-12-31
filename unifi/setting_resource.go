package unifi

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

var (
	_ resource.Resource                = &settingResource{}
	_ resource.ResourceWithImportState = &settingResource{}
)

func NewSettingResource() resource.Resource {
	return &settingResource{}
}

type settingResource struct {
	client *Client
}

type sshKeyModel struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Key     types.String `tfsdk:"key"`
	Comment types.String `tfsdk:"comment"`
}

type settingMgmtModel struct {
	AutoUpgrade types.Bool `tfsdk:"auto_upgrade"`
	SSHEnabled  types.Bool `tfsdk:"ssh_enabled"`
	SSHKeys     types.List `tfsdk:"ssh_keys"`
}

type settingRadiusModel struct {
	AccountingEnabled     types.Bool   `tfsdk:"accounting_enabled"`
	AcctPort              types.Int64  `tfsdk:"acct_port"`
	AuthPort              types.Int64  `tfsdk:"auth_port"`
	InterimUpdateInterval types.Int64  `tfsdk:"interim_update_interval"`
	Secret                types.String `tfsdk:"secret"`
}

type dnsVerificationModel struct {
	Domain             types.String `tfsdk:"domain"`
	PrimaryDNSServer   types.String `tfsdk:"primary_dns_server"`
	SecondaryDNSServer types.String `tfsdk:"secondary_dns_server"`
	SettingPreference  types.String `tfsdk:"setting_preference"`
}

type settingUSGModel struct {
	BroadcastPing                  types.Bool   `tfsdk:"broadcast_ping"`
	DNSVerification                types.Object `tfsdk:"dns_verification"`
	FtpModule                      types.Bool   `tfsdk:"ftp_module"`
	GeoIPFilteringBlock            types.String `tfsdk:"geo_ip_filtering_block"`
	GeoIPFilteringCountries        types.String `tfsdk:"geo_ip_filtering_countries"`
	GeoIPFilteringEnabled          types.Bool   `tfsdk:"geo_ip_filtering_enabled"`
	GeoIPFilteringTrafficDirection types.String `tfsdk:"geo_ip_filtering_traffic_direction"`
	GreModule                      types.Bool   `tfsdk:"gre_module"`
	H323Module                     types.Bool   `tfsdk:"h323_module"`
	ICMPTimeout                    types.Int64  `tfsdk:"icmp_timeout"`
	MssClamp                       types.String `tfsdk:"mss_clamp"`
	OffloadAccounting              types.Bool   `tfsdk:"offload_accounting"`
	OffloadL2Blocking              types.Bool   `tfsdk:"offload_l2_blocking"`
	OffloadSch                     types.Bool   `tfsdk:"offload_sch"`
	OtherTimeout                   types.Int64  `tfsdk:"other_timeout"`
	PptpModule                     types.Bool   `tfsdk:"pptp_module"`
	ReceiveRedirects               types.Bool   `tfsdk:"receive_redirects"`
	SendRedirects                  types.Bool   `tfsdk:"send_redirects"`
	SipModule                      types.Bool   `tfsdk:"sip_module"`
	SynCookies                     types.Bool   `tfsdk:"syn_cookies"`
	TCPCloseTimeout                types.Int64  `tfsdk:"tcp_close_timeout"`
	TCPCloseWaitTimeout            types.Int64  `tfsdk:"tcp_close_wait_timeout"`
	TCPEstablishedTimeout          types.Int64  `tfsdk:"tcp_established_timeout"`
	TCPFinWaitTimeout              types.Int64  `tfsdk:"tcp_fin_wait_timeout"`
	TCPLastAckTimeout              types.Int64  `tfsdk:"tcp_last_ack_timeout"`
	TCPSynRecvTimeout              types.Int64  `tfsdk:"tcp_syn_recv_timeout"`
	TCPSynSentTimeout              types.Int64  `tfsdk:"tcp_syn_sent_timeout"`
	TCPTimeWaitTimeout             types.Int64  `tfsdk:"tcp_time_wait_timeout"`
	TFTPModule                     types.Bool   `tfsdk:"tftp_module"`
	TimeoutSettingPreference       types.String `tfsdk:"timeout_setting_preference"`
	UDPOtherTimeout                types.Int64  `tfsdk:"udp_other_timeout"`
	UDPStreamTimeout               types.Int64  `tfsdk:"udp_stream_timeout"`
	UnbindWANMonitors              types.Bool   `tfsdk:"unbind_wan_monitors"`
	UpnpEnabled                    types.Bool   `tfsdk:"upnp_enabled"`
	UpnpNATPmpEnabled              types.Bool   `tfsdk:"upnp_nat_pmp_enabled"`
	UpnpSecureMode                 types.Bool   `tfsdk:"upnp_secure_mode"`
	UpnpWANInterface               types.String `tfsdk:"upnp_wan_interface"`
}

type settingResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Site   types.String `tfsdk:"site"`
	Mgmt   types.Object `tfsdk:"mgmt"`
	Radius types.Object `tfsdk:"radius"`
	USG    types.Object `tfsdk:"usg"`
}

func (r *settingResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_setting"
}

func (r *settingResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages settings for a UniFi site. Configure only the settings you need by providing the corresponding nested object.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the settings.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the settings with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mgmt": schema.SingleNestedAttribute{
				MarkdownDescription: "Management settings.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"auto_upgrade": schema.BoolAttribute{
						MarkdownDescription: "Automatically upgrade device firmware.",
						Optional:            true,
						Computed:            true,
					},
					"ssh_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable SSH authentication.",
						Optional:            true,
						Computed:            true,
					},
					"ssh_keys": schema.ListNestedAttribute{
						MarkdownDescription: "SSH keys.",
						Optional:            true,
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "Name of SSH key.",
									Required:            true,
								},
								"type": schema.StringAttribute{
									MarkdownDescription: "Type of SSH key, e.g. ssh-rsa.",
									Required:            true,
								},
								"key": schema.StringAttribute{
									MarkdownDescription: "Public SSH key.",
									Optional:            true,
									Computed:            true,
								},
								"comment": schema.StringAttribute{
									MarkdownDescription: "Comment.",
									Optional:            true,
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"radius": schema.SingleNestedAttribute{
				MarkdownDescription: "RADIUS settings.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"accounting_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable RADIUS accounting.",
						Optional:            true,
						Computed:            true,
					},
					"acct_port": schema.Int64Attribute{
						MarkdownDescription: "RADIUS accounting port.",
						Optional:            true,
						Computed:            true,
					},
					"auth_port": schema.Int64Attribute{
						MarkdownDescription: "RADIUS authentication port.",
						Optional:            true,
						Computed:            true,
					},
					"interim_update_interval": schema.Int64Attribute{
						MarkdownDescription: "Interim update interval in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"secret": schema.StringAttribute{
						MarkdownDescription: "RADIUS shared secret.",
						Optional:            true,
						Computed:            true,
						Sensitive:           true,
					},
				},
			},
			"usg": schema.SingleNestedAttribute{
				MarkdownDescription: "USG settings.",
				Optional:            true,
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"broadcast_ping": schema.BoolAttribute{
						MarkdownDescription: "Enable broadcast ping.",
						Optional:            true,
						Computed:            true,
					},
					"dns_verification": schema.SingleNestedAttribute{
						MarkdownDescription: "DNS verification settings.",
						Optional:            true,
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"domain": schema.StringAttribute{
								MarkdownDescription: "Domain for DNS verification.",
								Optional:            true,
								Computed:            true,
							},
							"primary_dns_server": schema.StringAttribute{
								MarkdownDescription: "Primary DNS server.",
								Optional:            true,
								Computed:            true,
							},
							"secondary_dns_server": schema.StringAttribute{
								MarkdownDescription: "Secondary DNS server.",
								Optional:            true,
								Computed:            true,
							},
							"setting_preference": schema.StringAttribute{
								MarkdownDescription: "Setting preference: auto or manual.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"ftp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable FTP module.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_block": schema.StringAttribute{
						MarkdownDescription: "Geo IP filtering action: block or allow.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_countries": schema.StringAttribute{
						MarkdownDescription: "Comma-separated list of country codes for geo IP filtering.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable geo IP filtering.",
						Optional:            true,
						Computed:            true,
					},
					"geo_ip_filtering_traffic_direction": schema.StringAttribute{
						MarkdownDescription: "Geo IP filtering traffic direction: both, ingress, or egress.",
						Optional:            true,
						Computed:            true,
					},
					"gre_module": schema.BoolAttribute{
						MarkdownDescription: "Enable GRE module.",
						Optional:            true,
						Computed:            true,
					},
					"h323_module": schema.BoolAttribute{
						MarkdownDescription: "Enable H.323 module.",
						Optional:            true,
						Computed:            true,
					},
					"icmp_timeout": schema.Int64Attribute{
						MarkdownDescription: "ICMP connection timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"mss_clamp": schema.StringAttribute{
						MarkdownDescription: "MSS clamping mode: auto, custom, or disabled.",
						Optional:            true,
						Computed:            true,
					},
					"offload_accounting": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for accounting.",
						Optional:            true,
						Computed:            true,
					},
					"offload_l2_blocking": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for L2 blocking.",
						Optional:            true,
						Computed:            true,
					},
					"offload_sch": schema.BoolAttribute{
						MarkdownDescription: "Enable hardware offload for scheduling.",
						Optional:            true,
						Computed:            true,
					},
					"other_timeout": schema.Int64Attribute{
						MarkdownDescription: "Other connections timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"pptp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable PPTP module.",
						Optional:            true,
						Computed:            true,
					},
					"receive_redirects": schema.BoolAttribute{
						MarkdownDescription: "Accept ICMP redirects.",
						Optional:            true,
						Computed:            true,
					},
					"send_redirects": schema.BoolAttribute{
						MarkdownDescription: "Send ICMP redirects.",
						Optional:            true,
						Computed:            true,
					},
					"sip_module": schema.BoolAttribute{
						MarkdownDescription: "Enable SIP module.",
						Optional:            true,
						Computed:            true,
					},
					"syn_cookies": schema.BoolAttribute{
						MarkdownDescription: "Enable SYN cookies.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_close_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP close timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_close_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP close wait timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_established_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP established connection timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_fin_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP fin wait timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_last_ack_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP last ACK timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_recv_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP SYN received timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_syn_sent_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP SYN sent timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tcp_time_wait_timeout": schema.Int64Attribute{
						MarkdownDescription: "TCP time wait timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"tftp_module": schema.BoolAttribute{
						MarkdownDescription: "Enable TFTP module.",
						Optional:            true,
						Computed:            true,
					},
					"timeout_setting_preference": schema.StringAttribute{
						MarkdownDescription: "Timeout setting preference: auto or manual.",
						Optional:            true,
						Computed:            true,
					},
					"udp_other_timeout": schema.Int64Attribute{
						MarkdownDescription: "UDP other timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"udp_stream_timeout": schema.Int64Attribute{
						MarkdownDescription: "UDP stream timeout in seconds.",
						Optional:            true,
						Computed:            true,
					},
					"unbind_wan_monitors": schema.BoolAttribute{
						MarkdownDescription: "Unbind WAN monitors.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_nat_pmp_enabled": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP NAT-PMP.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_secure_mode": schema.BoolAttribute{
						MarkdownDescription: "Enable UPnP secure mode.",
						Optional:            true,
						Computed:            true,
					},
					"upnp_wan_interface": schema.StringAttribute{
						MarkdownDescription: "UPnP WAN interface (e.g., WAN, WAN2).",
						Optional:            true,
						Computed:            true,
					},
				},
			},
		},
	}
}

func (r *settingResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)
		return
	}

	r.client = client
}

func (r *settingResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data settingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
	if !data.Mgmt.IsNull() && !data.Mgmt.IsUnknown() {
		var mgmt settingMgmtModel
		resp.Diagnostics.Append(data.Mgmt.As(ctx, &mgmt, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.mgmtModelToSetting(ctx, &mgmt)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Mgmt Setting", err.Error())
			return
		}
	}

	if !data.Radius.IsNull() && !data.Radius.IsUnknown() {
		var radius settingRadiusModel
		resp.Diagnostics.Append(data.Radius.As(ctx, &radius, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.radiusModelToSetting(ctx, &radius)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating Radius Setting", err.Error())
			return
		}
	}

	if !data.USG.IsNull() && !data.USG.IsUnknown() {
		var usg settingUSGModel
		resp.Diagnostics.Append(data.USG.As(ctx, &usg, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.usgModelToSetting(ctx, &usg)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Creating USG Setting", err.Error())
			return
		}
	}

	// Read back the settings
	r.readSettings(ctx, site, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data settingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	r.readSettings(ctx, site, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *settingResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var state settingResourceModel
	var plan settingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := state.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Update each configured setting type
	if !plan.Mgmt.IsNull() && !plan.Mgmt.IsUnknown() {
		var mgmt settingMgmtModel
		resp.Diagnostics.Append(plan.Mgmt.As(ctx, &mgmt, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.mgmtModelToSetting(ctx, &mgmt)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Mgmt Setting", err.Error())
			return
		}
	}

	if !plan.Radius.IsNull() && !plan.Radius.IsUnknown() {
		var radius settingRadiusModel
		resp.Diagnostics.Append(plan.Radius.As(ctx, &radius, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.radiusModelToSetting(ctx, &radius)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating Radius Setting", err.Error())
			return
		}
	}

	if !plan.USG.IsNull() && !plan.USG.IsUnknown() {
		var usg settingUSGModel
		resp.Diagnostics.Append(plan.USG.As(ctx, &usg, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}

		setting := r.usgModelToSetting(ctx, &usg)
		if err := r.client.UpdateSetting(ctx, site, setting); err != nil {
			resp.Diagnostics.AddError("Error Updating USG Setting", err.Error())
			return
		}
	}

	// Read back the settings
	r.readSettings(ctx, site, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *settingResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	// Settings cannot be deleted, only reset to defaults
	// Just remove from state
}

func (r *settingResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(
		ctx,
		path.Root("id"),
		req,
		resp,
	)
}

func (r *settingResource) readSettings(
	ctx context.Context,
	site string,
	data *settingResourceModel,
	diags *diag.Diagnostics,
) {
	// Set the ID to the site since settings are site-level
	data.ID = types.StringValue(site)
	data.Site = types.StringValue(site)

	// Only read settings that were configured in the plan, set others to null
	// Mgmt settings
	if !data.Mgmt.IsNull() && !data.Mgmt.IsUnknown() {
		// Get the current plan/state values
		var planMgmt settingMgmtModel
		diags.Append(data.Mgmt.As(ctx, &planMgmt, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, mgmtSetting, err := ui.GetSetting[*settings.Mgmt](r.client.Client, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Mgmt Setting", err.Error())
			return
		}

		mgmtModel := r.mgmtSettingToModel(ctx, mgmtSetting, &planMgmt)
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"auto_upgrade": types.BoolType,
			"ssh_enabled":  types.BoolType,
			"ssh_keys": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":    types.StringType,
						"type":    types.StringType,
						"key":     types.StringType,
						"comment": types.StringType,
					},
				},
			},
		}, mgmtModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Mgmt = objValue
	} else {
		data.Mgmt = types.ObjectNull(map[string]attr.Type{
			"auto_upgrade": types.BoolType,
			"ssh_enabled":  types.BoolType,
			"ssh_keys": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name":    types.StringType,
						"type":    types.StringType,
						"key":     types.StringType,
						"comment": types.StringType,
					},
				},
			},
		})
	}

	// Radius settings
	if !data.Radius.IsNull() && !data.Radius.IsUnknown() {
		// Get the current plan/state values
		var planRadius settingRadiusModel
		diags.Append(data.Radius.As(ctx, &planRadius, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, radiusSetting, err := ui.GetSetting[*settings.Radius](r.client.Client, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Radius Setting", err.Error())
			return
		}

		radiusModel := r.radiusSettingToModel(ctx, radiusSetting, &planRadius)
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"accounting_enabled":      types.BoolType,
			"acct_port":               types.Int64Type,
			"auth_port":               types.Int64Type,
			"interim_update_interval": types.Int64Type,
			"secret":                  types.StringType,
		}, radiusModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Radius = objValue
	} else {
		data.Radius = types.ObjectNull(map[string]attr.Type{
			"accounting_enabled":      types.BoolType,
			"acct_port":               types.Int64Type,
			"auth_port":               types.Int64Type,
			"interim_update_interval": types.Int64Type,
			"secret":                  types.StringType,
		})
	}

	// USG settings
	if !data.USG.IsNull() && !data.USG.IsUnknown() {
		// Get the current plan/state values
		var planUSG settingUSGModel
		diags.Append(data.USG.As(ctx, &planUSG, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		_, usgSetting, err := ui.GetSetting[*settings.Usg](r.client.Client, ctx, site)
		if err != nil {
			diags.AddError("Error Reading USG Setting", err.Error())
			return
		}

		usgModel := r.usgSettingToModel(ctx, usgSetting, &planUSG)
		objValue, d := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"broadcast_ping": types.BoolType,
			"dns_verification": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"domain":               types.StringType,
					"primary_dns_server":   types.StringType,
					"secondary_dns_server": types.StringType,
					"setting_preference":   types.StringType,
				},
			},
			"ftp_module":                         types.BoolType,
			"geo_ip_filtering_block":             types.StringType,
			"geo_ip_filtering_countries":         types.StringType,
			"geo_ip_filtering_enabled":           types.BoolType,
			"geo_ip_filtering_traffic_direction": types.StringType,
			"gre_module":                         types.BoolType,
			"h323_module":                        types.BoolType,
			"icmp_timeout":                       types.Int64Type,
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      types.Int64Type,
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  types.Int64Type,
			"tcp_close_wait_timeout":             types.Int64Type,
			"tcp_established_timeout":            types.Int64Type,
			"tcp_fin_wait_timeout":               types.Int64Type,
			"tcp_last_ack_timeout":               types.Int64Type,
			"tcp_syn_recv_timeout":               types.Int64Type,
			"tcp_syn_sent_timeout":               types.Int64Type,
			"tcp_time_wait_timeout":              types.Int64Type,
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  types.Int64Type,
			"udp_stream_timeout":                 types.Int64Type,
			"unbind_wan_monitors":                types.BoolType,
			"upnp_enabled":                       types.BoolType,
			"upnp_nat_pmp_enabled":               types.BoolType,
			"upnp_secure_mode":                   types.BoolType,
			"upnp_wan_interface":                 types.StringType,
		}, usgModel)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.USG = objValue
	} else {
		data.USG = types.ObjectNull(map[string]attr.Type{
			"broadcast_ping": types.BoolType,
			"dns_verification": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"domain":               types.StringType,
					"primary_dns_server":   types.StringType,
					"secondary_dns_server": types.StringType,
					"setting_preference":   types.StringType,
				},
			},
			"ftp_module":                         types.BoolType,
			"geo_ip_filtering_block":             types.StringType,
			"geo_ip_filtering_countries":         types.StringType,
			"geo_ip_filtering_enabled":           types.BoolType,
			"geo_ip_filtering_traffic_direction": types.StringType,
			"gre_module":                         types.BoolType,
			"h323_module":                        types.BoolType,
			"icmp_timeout":                       types.Int64Type,
			"mss_clamp":                          types.StringType,
			"offload_accounting":                 types.BoolType,
			"offload_l2_blocking":                types.BoolType,
			"offload_sch":                        types.BoolType,
			"other_timeout":                      types.Int64Type,
			"pptp_module":                        types.BoolType,
			"receive_redirects":                  types.BoolType,
			"send_redirects":                     types.BoolType,
			"sip_module":                         types.BoolType,
			"syn_cookies":                        types.BoolType,
			"tcp_close_timeout":                  types.Int64Type,
			"tcp_close_wait_timeout":             types.Int64Type,
			"tcp_established_timeout":            types.Int64Type,
			"tcp_fin_wait_timeout":               types.Int64Type,
			"tcp_last_ack_timeout":               types.Int64Type,
			"tcp_syn_recv_timeout":               types.Int64Type,
			"tcp_syn_sent_timeout":               types.Int64Type,
			"tcp_time_wait_timeout":              types.Int64Type,
			"tftp_module":                        types.BoolType,
			"timeout_setting_preference":         types.StringType,
			"udp_other_timeout":                  types.Int64Type,
			"udp_stream_timeout":                 types.Int64Type,
			"unbind_wan_monitors":                types.BoolType,
			"upnp_enabled":                       types.BoolType,
			"upnp_nat_pmp_enabled":               types.BoolType,
			"upnp_secure_mode":                   types.BoolType,
			"upnp_wan_interface":                 types.StringType,
		})
	}
}

// Mgmt conversion functions.
func (r *settingResource) mgmtModelToSetting(
	ctx context.Context,
	model *settingMgmtModel,
) *settings.Mgmt {
	setting := &settings.Mgmt{}

	if !model.AutoUpgrade.IsNull() {
		setting.AutoUpgrade = model.AutoUpgrade.ValueBool()
	}
	if !model.SSHEnabled.IsNull() {
		setting.XSshEnabled = model.SSHEnabled.ValueBool()
	}

	if !model.SSHKeys.IsNull() && !model.SSHKeys.IsUnknown() {
		var sshKeys []sshKeyModel
		model.SSHKeys.ElementsAs(ctx, &sshKeys, false)
		for _, sshKey := range sshKeys {
			setting.XSshKeys = append(setting.XSshKeys, settings.SettingMgmtXSshKeys{
				Name:    sshKey.Name.ValueString(),
				KeyType: sshKey.Type.ValueString(),
				Key:     sshKey.Key.ValueString(),
				Comment: sshKey.Comment.ValueString(),
			})
		}
	}

	return setting
}

func (r *settingResource) mgmtSettingToModel(
	ctx context.Context,
	setting *settings.Mgmt,
	plan *settingMgmtModel,
) *settingMgmtModel {
	model := &settingMgmtModel{}

	// Only populate fields that were explicitly configured in the plan
	if !plan.AutoUpgrade.IsNull() && !plan.AutoUpgrade.IsUnknown() {
		model.AutoUpgrade = types.BoolValue(setting.AutoUpgrade)
	} else {
		model.AutoUpgrade = types.BoolNull()
	}

	if !plan.SSHEnabled.IsNull() && !plan.SSHEnabled.IsUnknown() {
		model.SSHEnabled = types.BoolValue(setting.XSshEnabled)
	} else {
		model.SSHEnabled = types.BoolNull()
	}

	if !plan.SSHKeys.IsNull() && !plan.SSHKeys.IsUnknown() {
		if len(setting.XSshKeys) > 0 {
			var sshKeys []sshKeyModel
			for _, sshKey := range setting.XSshKeys {
				sshKeys = append(sshKeys, sshKeyModel{
					Name:    types.StringValue(sshKey.Name),
					Type:    types.StringValue(sshKey.KeyType),
					Key:     types.StringValue(sshKey.Key),
					Comment: types.StringValue(sshKey.Comment),
				})
			}
			listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":    types.StringType,
					"type":    types.StringType,
					"key":     types.StringType,
					"comment": types.StringType,
				},
			}, sshKeys)
			model.SSHKeys = listValue
		} else {
			model.SSHKeys = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":    types.StringType,
					"type":    types.StringType,
					"key":     types.StringType,
					"comment": types.StringType,
				},
			})
		}
	} else {
		model.SSHKeys = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":    types.StringType,
				"type":    types.StringType,
				"key":     types.StringType,
				"comment": types.StringType,
			},
		})
	}

	return model
}

// Radius conversion functions.
func (r *settingResource) radiusModelToSetting(
	_ context.Context,
	model *settingRadiusModel,
) *settings.Radius {
	setting := &settings.Radius{}

	if !model.AccountingEnabled.IsNull() {
		setting.AccountingEnabled = model.AccountingEnabled.ValueBool()
	}
	if !model.AcctPort.IsNull() {
		setting.AcctPort = int(model.AcctPort.ValueInt64())
	}
	if !model.AuthPort.IsNull() {
		setting.AuthPort = int(model.AuthPort.ValueInt64())
	}
	if !model.InterimUpdateInterval.IsNull() {
		setting.InterimUpdateInterval = int(model.InterimUpdateInterval.ValueInt64())
	}
	if !model.Secret.IsNull() {
		setting.XSecret = model.Secret.ValueString()
	}

	return setting
}

func (r *settingResource) radiusSettingToModel(
	_ context.Context,
	setting *settings.Radius,
	plan *settingRadiusModel,
) *settingRadiusModel {
	model := &settingRadiusModel{}

	// Only populate fields that were explicitly configured in the plan
	if !plan.AccountingEnabled.IsNull() && !plan.AccountingEnabled.IsUnknown() {
		model.AccountingEnabled = types.BoolValue(setting.AccountingEnabled)
	} else {
		model.AccountingEnabled = types.BoolNull()
	}

	if !plan.AcctPort.IsNull() && !plan.AcctPort.IsUnknown() {
		model.AcctPort = types.Int64Value(int64(setting.AcctPort))
	} else {
		model.AcctPort = types.Int64Null()
	}

	if !plan.AuthPort.IsNull() && !plan.AuthPort.IsUnknown() {
		model.AuthPort = types.Int64Value(int64(setting.AuthPort))
	} else {
		model.AuthPort = types.Int64Null()
	}

	if !plan.InterimUpdateInterval.IsNull() && !plan.InterimUpdateInterval.IsUnknown() {
		model.InterimUpdateInterval = types.Int64Value(int64(setting.InterimUpdateInterval))
	} else {
		model.InterimUpdateInterval = types.Int64Null()
	}

	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() {
		if setting.XSecret != "" {
			model.Secret = types.StringValue(setting.XSecret)
		} else {
			model.Secret = types.StringNull()
		}
	} else {
		model.Secret = types.StringNull()
	}

	return model
}

// USG conversion functions.
func (r *settingResource) usgModelToSetting(
	ctx context.Context,
	model *settingUSGModel,
) *settings.Usg {
	setting := &settings.Usg{}

	if !model.BroadcastPing.IsNull() {
		setting.BroadcastPing = model.BroadcastPing.ValueBool()
	}
	if !model.DNSVerification.IsNull() && !model.DNSVerification.IsUnknown() {
		var dnsVerif dnsVerificationModel
		model.DNSVerification.As(ctx, &dnsVerif, basetypes.ObjectAsOptions{})
		setting.DNSVerification = settings.SettingUsgDNSVerification{
			Domain:             dnsVerif.Domain.ValueString(),
			PrimaryDNSServer:   dnsVerif.PrimaryDNSServer.ValueString(),
			SecondaryDNSServer: dnsVerif.SecondaryDNSServer.ValueString(),
			SettingPreference:  dnsVerif.SettingPreference.ValueString(),
		}
	}
	if !model.FtpModule.IsNull() {
		setting.FtpModule = model.FtpModule.ValueBool()
	}
	if !model.GeoIPFilteringBlock.IsNull() {
		setting.GeoIPFilteringBlock = model.GeoIPFilteringBlock.ValueString()
	}
	if !model.GeoIPFilteringCountries.IsNull() {
		setting.GeoIPFilteringCountries = model.GeoIPFilteringCountries.ValueString()
	}
	if !model.GeoIPFilteringEnabled.IsNull() {
		setting.GeoIPFilteringEnabled = model.GeoIPFilteringEnabled.ValueBool()
	}
	if !model.GeoIPFilteringTrafficDirection.IsNull() {
		setting.GeoIPFilteringTrafficDirection = model.GeoIPFilteringTrafficDirection.ValueString()
	}
	if !model.GreModule.IsNull() {
		setting.GreModule = model.GreModule.ValueBool()
	}
	if !model.H323Module.IsNull() {
		setting.H323Module = model.H323Module.ValueBool()
	}
	if !model.ICMPTimeout.IsNull() {
		setting.ICMPTimeout = int(model.ICMPTimeout.ValueInt64())
	}
	if !model.MssClamp.IsNull() {
		setting.MssClamp = model.MssClamp.ValueString()
	}
	if !model.OffloadAccounting.IsNull() {
		setting.OffloadAccounting = model.OffloadAccounting.ValueBool()
	}
	if !model.OffloadL2Blocking.IsNull() {
		setting.OffloadL2Blocking = model.OffloadL2Blocking.ValueBool()
	}
	if !model.OffloadSch.IsNull() {
		setting.OffloadSch = model.OffloadSch.ValueBool()
	}
	if !model.OtherTimeout.IsNull() {
		setting.OtherTimeout = int(model.OtherTimeout.ValueInt64())
	}
	if !model.PptpModule.IsNull() {
		setting.PptpModule = model.PptpModule.ValueBool()
	}
	if !model.ReceiveRedirects.IsNull() {
		setting.ReceiveRedirects = model.ReceiveRedirects.ValueBool()
	}
	if !model.SendRedirects.IsNull() {
		setting.SendRedirects = model.SendRedirects.ValueBool()
	}
	if !model.SipModule.IsNull() {
		setting.SipModule = model.SipModule.ValueBool()
	}
	if !model.SynCookies.IsNull() {
		setting.SynCookies = model.SynCookies.ValueBool()
	}
	if !model.TCPCloseTimeout.IsNull() {
		setting.TCPCloseTimeout = int(model.TCPCloseTimeout.ValueInt64())
	}
	if !model.TCPCloseWaitTimeout.IsNull() {
		setting.TCPCloseWaitTimeout = int(model.TCPCloseWaitTimeout.ValueInt64())
	}
	if !model.TCPEstablishedTimeout.IsNull() {
		setting.TCPEstablishedTimeout = int(model.TCPEstablishedTimeout.ValueInt64())
	}
	if !model.TCPFinWaitTimeout.IsNull() {
		setting.TCPFinWaitTimeout = int(model.TCPFinWaitTimeout.ValueInt64())
	}
	if !model.TCPLastAckTimeout.IsNull() {
		setting.TCPLastAckTimeout = int(model.TCPLastAckTimeout.ValueInt64())
	}
	if !model.TCPSynRecvTimeout.IsNull() {
		setting.TCPSynRecvTimeout = int(model.TCPSynRecvTimeout.ValueInt64())
	}
	if !model.TCPSynSentTimeout.IsNull() {
		setting.TCPSynSentTimeout = int(model.TCPSynSentTimeout.ValueInt64())
	}
	if !model.TCPTimeWaitTimeout.IsNull() {
		setting.TCPTimeWaitTimeout = int(model.TCPTimeWaitTimeout.ValueInt64())
	}
	if !model.TFTPModule.IsNull() {
		setting.TFTPModule = model.TFTPModule.ValueBool()
	}
	if !model.TimeoutSettingPreference.IsNull() {
		setting.TimeoutSettingPreference = model.TimeoutSettingPreference.ValueString()
	}
	if !model.UDPOtherTimeout.IsNull() {
		setting.UDPOtherTimeout = int(model.UDPOtherTimeout.ValueInt64())
	}
	if !model.UDPStreamTimeout.IsNull() {
		setting.UDPStreamTimeout = int(model.UDPStreamTimeout.ValueInt64())
	}
	if !model.UnbindWANMonitors.IsNull() {
		setting.UnbindWANMonitors = model.UnbindWANMonitors.ValueBool()
	}
	if !model.UpnpEnabled.IsNull() {
		setting.UpnpEnabled = model.UpnpEnabled.ValueBool()
	}
	if !model.UpnpNATPmpEnabled.IsNull() {
		setting.UpnpNATPmpEnabled = model.UpnpNATPmpEnabled.ValueBool()
	}
	if !model.UpnpSecureMode.IsNull() {
		setting.UpnpSecureMode = model.UpnpSecureMode.ValueBool()
	}
	if !model.UpnpWANInterface.IsNull() {
		setting.UpnpWANInterface = model.UpnpWANInterface.ValueString()
	}

	return setting
}

func (r *settingResource) usgSettingToModel(
	ctx context.Context,
	setting *settings.Usg,
	plan *settingUSGModel,
) *settingUSGModel {
	model := &settingUSGModel{}

	// Only populate fields that were explicitly configured in the plan
	if !plan.BroadcastPing.IsNull() && !plan.BroadcastPing.IsUnknown() {
		model.BroadcastPing = types.BoolValue(setting.BroadcastPing)
	} else {
		model.BroadcastPing = types.BoolNull()
	}

	if !plan.DNSVerification.IsNull() && !plan.DNSVerification.IsUnknown() {
		dnsVerif := dnsVerificationModel{
			Domain:             types.StringValue(setting.DNSVerification.Domain),
			PrimaryDNSServer:   types.StringValue(setting.DNSVerification.PrimaryDNSServer),
			SecondaryDNSServer: types.StringValue(setting.DNSVerification.SecondaryDNSServer),
			SettingPreference:  types.StringValue(setting.DNSVerification.SettingPreference),
		}
		objValue, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"domain":               types.StringType,
			"primary_dns_server":   types.StringType,
			"secondary_dns_server": types.StringType,
			"setting_preference":   types.StringType,
		}, dnsVerif)
		model.DNSVerification = objValue
	} else {
		model.DNSVerification = types.ObjectNull(map[string]attr.Type{
			"domain":               types.StringType,
			"primary_dns_server":   types.StringType,
			"secondary_dns_server": types.StringType,
			"setting_preference":   types.StringType,
		})
	}

	if !plan.FtpModule.IsNull() && !plan.FtpModule.IsUnknown() {
		model.FtpModule = types.BoolValue(setting.FtpModule)
	} else {
		model.FtpModule = types.BoolNull()
	}

	if !plan.GeoIPFilteringBlock.IsNull() && !plan.GeoIPFilteringBlock.IsUnknown() {
		if setting.GeoIPFilteringBlock != "" {
			model.GeoIPFilteringBlock = types.StringValue(setting.GeoIPFilteringBlock)
		} else {
			model.GeoIPFilteringBlock = types.StringNull()
		}
	} else {
		model.GeoIPFilteringBlock = types.StringNull()
	}

	if !plan.GeoIPFilteringCountries.IsNull() && !plan.GeoIPFilteringCountries.IsUnknown() {
		if setting.GeoIPFilteringCountries != "" {
			model.GeoIPFilteringCountries = types.StringValue(setting.GeoIPFilteringCountries)
		} else {
			model.GeoIPFilteringCountries = types.StringNull()
		}
	} else {
		model.GeoIPFilteringCountries = types.StringNull()
	}

	if !plan.GeoIPFilteringEnabled.IsNull() && !plan.GeoIPFilteringEnabled.IsUnknown() {
		model.GeoIPFilteringEnabled = types.BoolValue(setting.GeoIPFilteringEnabled)
	} else {
		model.GeoIPFilteringEnabled = types.BoolNull()
	}

	if !plan.GeoIPFilteringTrafficDirection.IsNull() &&
		!plan.GeoIPFilteringTrafficDirection.IsUnknown() {
		if setting.GeoIPFilteringTrafficDirection != "" {
			model.GeoIPFilteringTrafficDirection = types.StringValue(
				setting.GeoIPFilteringTrafficDirection,
			)
		} else {
			model.GeoIPFilteringTrafficDirection = types.StringNull()
		}
	} else {
		model.GeoIPFilteringTrafficDirection = types.StringNull()
	}

	if !plan.GreModule.IsNull() && !plan.GreModule.IsUnknown() {
		model.GreModule = types.BoolValue(setting.GreModule)
	} else {
		model.GreModule = types.BoolNull()
	}

	if !plan.H323Module.IsNull() && !plan.H323Module.IsUnknown() {
		model.H323Module = types.BoolValue(setting.H323Module)
	} else {
		model.H323Module = types.BoolNull()
	}

	if !plan.ICMPTimeout.IsNull() && !plan.ICMPTimeout.IsUnknown() {
		model.ICMPTimeout = types.Int64Value(int64(setting.ICMPTimeout))
	} else {
		model.ICMPTimeout = types.Int64Null()
	}

	if !plan.MssClamp.IsNull() && !plan.MssClamp.IsUnknown() {
		if setting.MssClamp != "" {
			model.MssClamp = types.StringValue(setting.MssClamp)
		} else {
			model.MssClamp = types.StringNull()
		}
	} else {
		model.MssClamp = types.StringNull()
	}

	if !plan.OffloadAccounting.IsNull() && !plan.OffloadAccounting.IsUnknown() {
		model.OffloadAccounting = types.BoolValue(setting.OffloadAccounting)
	} else {
		model.OffloadAccounting = types.BoolNull()
	}

	if !plan.OffloadL2Blocking.IsNull() && !plan.OffloadL2Blocking.IsUnknown() {
		model.OffloadL2Blocking = types.BoolValue(setting.OffloadL2Blocking)
	} else {
		model.OffloadL2Blocking = types.BoolNull()
	}

	if !plan.OffloadSch.IsNull() && !plan.OffloadSch.IsUnknown() {
		model.OffloadSch = types.BoolValue(setting.OffloadSch)
	} else {
		model.OffloadSch = types.BoolNull()
	}

	if !plan.OtherTimeout.IsNull() && !plan.OtherTimeout.IsUnknown() {
		model.OtherTimeout = types.Int64Value(int64(setting.OtherTimeout))
	} else {
		model.OtherTimeout = types.Int64Null()
	}

	if !plan.PptpModule.IsNull() && !plan.PptpModule.IsUnknown() {
		model.PptpModule = types.BoolValue(setting.PptpModule)
	} else {
		model.PptpModule = types.BoolNull()
	}

	if !plan.ReceiveRedirects.IsNull() && !plan.ReceiveRedirects.IsUnknown() {
		model.ReceiveRedirects = types.BoolValue(setting.ReceiveRedirects)
	} else {
		model.ReceiveRedirects = types.BoolNull()
	}

	if !plan.SendRedirects.IsNull() && !plan.SendRedirects.IsUnknown() {
		model.SendRedirects = types.BoolValue(setting.SendRedirects)
	} else {
		model.SendRedirects = types.BoolNull()
	}

	if !plan.SipModule.IsNull() && !plan.SipModule.IsUnknown() {
		model.SipModule = types.BoolValue(setting.SipModule)
	} else {
		model.SipModule = types.BoolNull()
	}

	if !plan.SynCookies.IsNull() && !plan.SynCookies.IsUnknown() {
		model.SynCookies = types.BoolValue(setting.SynCookies)
	} else {
		model.SynCookies = types.BoolNull()
	}

	if !plan.TCPCloseTimeout.IsNull() && !plan.TCPCloseTimeout.IsUnknown() {
		model.TCPCloseTimeout = types.Int64Value(int64(setting.TCPCloseTimeout))
	} else {
		model.TCPCloseTimeout = types.Int64Null()
	}

	if !plan.TCPCloseWaitTimeout.IsNull() && !plan.TCPCloseWaitTimeout.IsUnknown() {
		model.TCPCloseWaitTimeout = types.Int64Value(int64(setting.TCPCloseWaitTimeout))
	} else {
		model.TCPCloseWaitTimeout = types.Int64Null()
	}

	if !plan.TCPEstablishedTimeout.IsNull() && !plan.TCPEstablishedTimeout.IsUnknown() {
		model.TCPEstablishedTimeout = types.Int64Value(int64(setting.TCPEstablishedTimeout))
	} else {
		model.TCPEstablishedTimeout = types.Int64Null()
	}

	if !plan.TCPFinWaitTimeout.IsNull() && !plan.TCPFinWaitTimeout.IsUnknown() {
		model.TCPFinWaitTimeout = types.Int64Value(int64(setting.TCPFinWaitTimeout))
	} else {
		model.TCPFinWaitTimeout = types.Int64Null()
	}

	if !plan.TCPLastAckTimeout.IsNull() && !plan.TCPLastAckTimeout.IsUnknown() {
		model.TCPLastAckTimeout = types.Int64Value(int64(setting.TCPLastAckTimeout))
	} else {
		model.TCPLastAckTimeout = types.Int64Null()
	}

	if !plan.TCPSynRecvTimeout.IsNull() && !plan.TCPSynRecvTimeout.IsUnknown() {
		model.TCPSynRecvTimeout = types.Int64Value(int64(setting.TCPSynRecvTimeout))
	} else {
		model.TCPSynRecvTimeout = types.Int64Null()
	}

	if !plan.TCPSynSentTimeout.IsNull() && !plan.TCPSynSentTimeout.IsUnknown() {
		model.TCPSynSentTimeout = types.Int64Value(int64(setting.TCPSynSentTimeout))
	} else {
		model.TCPSynSentTimeout = types.Int64Null()
	}

	if !plan.TCPTimeWaitTimeout.IsNull() && !plan.TCPTimeWaitTimeout.IsUnknown() {
		model.TCPTimeWaitTimeout = types.Int64Value(int64(setting.TCPTimeWaitTimeout))
	} else {
		model.TCPTimeWaitTimeout = types.Int64Null()
	}

	if !plan.TFTPModule.IsNull() && !plan.TFTPModule.IsUnknown() {
		model.TFTPModule = types.BoolValue(setting.TFTPModule)
	} else {
		model.TFTPModule = types.BoolNull()
	}

	if !plan.TimeoutSettingPreference.IsNull() && !plan.TimeoutSettingPreference.IsUnknown() {
		if setting.TimeoutSettingPreference != "" {
			model.TimeoutSettingPreference = types.StringValue(setting.TimeoutSettingPreference)
		} else {
			model.TimeoutSettingPreference = types.StringNull()
		}
	} else {
		model.TimeoutSettingPreference = types.StringNull()
	}

	if !plan.UDPOtherTimeout.IsNull() && !plan.UDPOtherTimeout.IsUnknown() {
		model.UDPOtherTimeout = types.Int64Value(int64(setting.UDPOtherTimeout))
	} else {
		model.UDPOtherTimeout = types.Int64Null()
	}

	if !plan.UDPStreamTimeout.IsNull() && !plan.UDPStreamTimeout.IsUnknown() {
		model.UDPStreamTimeout = types.Int64Value(int64(setting.UDPStreamTimeout))
	} else {
		model.UDPStreamTimeout = types.Int64Null()
	}

	if !plan.UnbindWANMonitors.IsNull() && !plan.UnbindWANMonitors.IsUnknown() {
		model.UnbindWANMonitors = types.BoolValue(setting.UnbindWANMonitors)
	} else {
		model.UnbindWANMonitors = types.BoolNull()
	}

	if !plan.UpnpEnabled.IsNull() && !plan.UpnpEnabled.IsUnknown() {
		model.UpnpEnabled = types.BoolValue(setting.UpnpEnabled)
	} else {
		model.UpnpEnabled = types.BoolNull()
	}

	if !plan.UpnpNATPmpEnabled.IsNull() && !plan.UpnpNATPmpEnabled.IsUnknown() {
		model.UpnpNATPmpEnabled = types.BoolValue(setting.UpnpNATPmpEnabled)
	} else {
		model.UpnpNATPmpEnabled = types.BoolNull()
	}

	if !plan.UpnpSecureMode.IsNull() && !plan.UpnpSecureMode.IsUnknown() {
		model.UpnpSecureMode = types.BoolValue(setting.UpnpSecureMode)
	} else {
		model.UpnpSecureMode = types.BoolNull()
	}

	if !plan.UpnpWANInterface.IsNull() && !plan.UpnpWANInterface.IsUnknown() {
		if setting.UpnpWANInterface != "" {
			model.UpnpWANInterface = types.StringValue(setting.UpnpWANInterface)
		} else {
			model.UpnpWANInterface = types.StringNull()
		}
	} else {
		model.UpnpWANInterface = types.StringNull()
	}

	return model
}
