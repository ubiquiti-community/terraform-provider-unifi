package unifi

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// The mdns and radio_ai blocks of unifi_setting. Both write by read-modify-write:
// the current setting is fetched and only the attributes set in config are
// overlaid, so keys this provider does not model keep their controller values.

// --- mdns --------------------------------------------------------------------

type settingMdnsModel struct {
	Mode                 types.String `tfsdk:"mode"`
	EnabledFor           types.String `tfsdk:"enabled_for"`
	EnabledForNetworkIDs types.List   `tfsdk:"enabled_for_network_ids"`
	PredefinedServices   types.List   `tfsdk:"predefined_services"`
	CustomServices       types.List   `tfsdk:"custom_services"`
}

type settingMdnsCustomServiceModel struct {
	Name    types.String `tfsdk:"name"`
	Address types.String `tfsdk:"address"`
}

var (
	mdnsCustomServiceAttrTypes = map[string]attr.Type{
		"name":    types.StringType,
		"address": types.StringType,
	}
	mdnsAttrTypes = map[string]attr.Type{
		"mode":                    types.StringType,
		"enabled_for":             types.StringType,
		"enabled_for_network_ids": types.ListType{ElemType: types.StringType},
		"predefined_services":     types.ListType{ElemType: types.StringType},
		"custom_services": types.ListType{
			ElemType: types.ObjectType{AttrTypes: mdnsCustomServiceAttrTypes},
		},
	}
)

func mdnsSchemaAttribute() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Multicast DNS (Settings > Networks > Multicast DNS): which networks mDNS is " +
			"reflected between, and which services. Options not exposed by this block are preserved across updates.",
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				MarkdownDescription: "Which services are reflected: `all`, `auto`, or `custom` " +
					"(only `predefined_services` and `custom_services`).",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("all", "auto", "custom")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled_for": schema.StringAttribute{
				MarkdownDescription: "Networks mDNS is enabled for: `all`, `some` (the networks in " +
					"`enabled_for_network_ids`), or `none`.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("all", "some", "none")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled_for_network_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the networks mDNS is enabled for when `enabled_for` is `some`.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"predefined_services": schema.ListAttribute{
				MarkdownDescription: "Codes of the predefined services reflected in `custom` mode, e.g. " +
					"`google_chromecast`, `apple_airPlay`, `printers`.",
				ElementType:   types.StringType,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"custom_services": schema.ListNestedAttribute{
				MarkdownDescription: "Additional services reflected in `custom` mode.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "Display name of the service.",
							Optional:            true,
						},
						"address": schema.StringAttribute{
							MarkdownDescription: "Service type, e.g. `_myservice._tcp.local`.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *settingResource) mdnsModelToSetting(
	ctx context.Context,
	m *settingMdnsModel,
	base *settings.Mdns,
	diags *diag.Diagnostics,
) *settings.Mdns {
	s := base
	if isKnown(m.Mode) {
		s.Mode = m.Mode.ValueString()
	}
	if isKnown(m.EnabledFor) {
		s.EnabledFor = m.EnabledFor.ValueString()
	}
	if isKnown(m.EnabledForNetworkIDs) {
		var ids []string
		diags.Append(m.EnabledForNetworkIDs.ElementsAs(ctx, &ids, false)...)
		s.EnabledForNetworkIDs = ids
	}
	if isKnown(m.PredefinedServices) {
		var codes []string
		diags.Append(m.PredefinedServices.ElementsAs(ctx, &codes, false)...)
		s.PredefinedServices = make([]settings.SettingMdnsPredefinedServices, len(codes))
		for i, c := range codes {
			s.PredefinedServices[i] = settings.SettingMdnsPredefinedServices{Code: c}
		}
	}
	if isKnown(m.CustomServices) {
		var svcs []settingMdnsCustomServiceModel
		diags.Append(m.CustomServices.ElementsAs(ctx, &svcs, false)...)
		s.CustomServices = make([]settings.SettingMdnsCustomServices, len(svcs))
		for i, c := range svcs {
			s.CustomServices[i] = settings.SettingMdnsCustomServices{
				Name:    c.Name.ValueString(),
				Address: c.Address.ValueString(),
			}
		}
	}
	return s
}

func (r *settingResource) mdnsSettingToModel(
	ctx context.Context,
	s *settings.Mdns,
	diags *diag.Diagnostics,
) settingMdnsModel {
	m := settingMdnsModel{
		Mode:       types.StringValue(s.Mode),
		EnabledFor: types.StringValue(s.EnabledFor),
	}
	var d diag.Diagnostics
	m.EnabledForNetworkIDs, d = types.ListValueFrom(ctx, types.StringType, nonNilStrings(s.EnabledForNetworkIDs))
	diags.Append(d...)

	codes := make([]string, len(s.PredefinedServices))
	for i, p := range s.PredefinedServices {
		codes[i] = p.Code
	}
	m.PredefinedServices, d = types.ListValueFrom(ctx, types.StringType, codes)
	diags.Append(d...)

	svcs := make([]settingMdnsCustomServiceModel, len(s.CustomServices))
	for i, c := range s.CustomServices {
		svcs[i] = settingMdnsCustomServiceModel{
			Name:    stringOrNull(c.Name),
			Address: types.StringValue(c.Address),
		}
	}
	m.CustomServices, d = types.ListValueFrom(
		ctx, types.ObjectType{AttrTypes: mdnsCustomServiceAttrTypes}, svcs)
	diags.Append(d...)
	return m
}

// --- radio_ai ----------------------------------------------------------------

type settingRadioAiModel struct {
	Enabled                     types.Bool   `tfsdk:"enabled"`
	AutoEnabled                 types.Bool   `tfsdk:"auto_enabled"`
	CronExpr                    types.String `tfsdk:"cron_expr"`
	SettingPreference           types.String `tfsdk:"setting_preference"`
	AutoChannelPresetsType      types.String `tfsdk:"auto_channel_presets_type"`
	AutoAdjustChannelsToCountry types.Bool   `tfsdk:"auto_adjust_channels_to_country"`
	Optimize                    types.List   `tfsdk:"optimize"`
	Radios                      types.List   `tfsdk:"radios"`
	ChannelsNg                  types.List   `tfsdk:"channels_ng"`
	ChannelsNa                  types.List   `tfsdk:"channels_na"`
	Channels6e                  types.List   `tfsdk:"channels_6e"`
	HtModesNg                   types.List   `tfsdk:"ht_modes_ng"`
	HtModesNa                   types.List   `tfsdk:"ht_modes_na"`
	ExcludeDevices              types.List   `tfsdk:"exclude_devices"`
	HighPriorityDevices         types.List   `tfsdk:"high_priority_devices"`
	ChannelsBlacklist           types.List   `tfsdk:"channels_blacklist"`
	RadiosConfiguration         types.List   `tfsdk:"radios_configuration"`
}

type settingRadioAiBlacklistModel struct {
	Radio        types.String `tfsdk:"radio"`
	Channel      types.Int64  `tfsdk:"channel"`
	ChannelWidth types.Int64  `tfsdk:"channel_width"`
}

type settingRadioAiRadioConfigModel struct {
	Radio        types.String `tfsdk:"radio"`
	ChannelWidth types.Int64  `tfsdk:"channel_width"`
	Dfs          types.Bool   `tfsdk:"dfs"`
}

var (
	radioAiBlacklistAttrTypes = map[string]attr.Type{
		"radio":         types.StringType,
		"channel":       types.Int64Type,
		"channel_width": types.Int64Type,
	}
	radioAiRadioConfigAttrTypes = map[string]attr.Type{
		"radio":         types.StringType,
		"channel_width": types.Int64Type,
		"dfs":           types.BoolType,
	}
	radioAiAttrTypes = map[string]attr.Type{
		"enabled":                         types.BoolType,
		"auto_enabled":                    types.BoolType,
		"cron_expr":                       types.StringType,
		"setting_preference":              types.StringType,
		"auto_channel_presets_type":       types.StringType,
		"auto_adjust_channels_to_country": types.BoolType,
		"optimize":                        types.ListType{ElemType: types.StringType},
		"radios":                          types.ListType{ElemType: types.StringType},
		"channels_ng":                     types.ListType{ElemType: types.Int64Type},
		"channels_na":                     types.ListType{ElemType: types.Int64Type},
		"channels_6e":                     types.ListType{ElemType: types.Int64Type},
		"ht_modes_ng":                     types.ListType{ElemType: types.Int64Type},
		"ht_modes_na":                     types.ListType{ElemType: types.Int64Type},
		"exclude_devices":                 types.ListType{ElemType: types.StringType},
		"high_priority_devices":           types.ListType{ElemType: types.StringType},
		"channels_blacklist": types.ListType{
			ElemType: types.ObjectType{AttrTypes: radioAiBlacklistAttrTypes},
		},
		"radios_configuration": types.ListType{
			ElemType: types.ObjectType{AttrTypes: radioAiRadioConfigAttrTypes},
		},
	}
)

func radioAiSchemaAttribute() schema.Attribute {
	optStr := func(desc string, v ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: desc, Optional: true, Computed: true, Validators: v,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	optBool := func(desc string) schema.BoolAttribute {
		return schema.BoolAttribute{
			MarkdownDescription: desc, Optional: true, Computed: true,
			PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		}
	}
	optList := func(desc string, elem attr.Type) schema.ListAttribute {
		return schema.ListAttribute{
			MarkdownDescription: desc, ElementType: elem, Optional: true, Computed: true,
			PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
		}
	}
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Radio AI / Nightly Channel Optimization (Settings > WiFi > Radio Manager): " +
			"when the controller re-plans channels and which channels and widths it may use. " +
			"Options not exposed by this block are preserved across updates.",
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"enabled":      optBool("Whether Radio AI is enabled."),
			"auto_enabled": optBool("Whether the scheduled (nightly) optimization runs."),
			"cron_expr":    optStr("Cron expression for the scheduled optimization, e.g. `0 4 * * *`."),
			"setting_preference": optStr("`auto` or `manual`.",
				stringvalidator.OneOf("auto", "manual")),
			"auto_channel_presets_type": optStr(
				"Channel preset: `maximum_speed`, `conservative` or `custom` (the `channels_*` lists).",
				stringvalidator.OneOf("maximum_speed", "conservative", "custom")),
			"auto_adjust_channels_to_country": optBool("Restrict the channel lists to the site's country."),
			"optimize":                        optList("What is optimized: `channel` and/or `power`.", types.StringType),
			"radios":                          optList("Radio bands included: `ng`, `na`, `6e`.", types.StringType),
			"channels_ng":                     optList("2.4 GHz channels Radio AI may use.", types.Int64Type),
			"channels_na":                     optList("5 GHz channels Radio AI may use.", types.Int64Type),
			"channels_6e":                     optList("6 GHz channels Radio AI may use.", types.Int64Type),
			"ht_modes_ng":                     optList("2.4 GHz channel widths (MHz) Radio AI may use.", types.Int64Type),
			"ht_modes_na":                     optList("5 GHz channel widths (MHz) Radio AI may use.", types.Int64Type),
			"exclude_devices": optList(
				"MAC addresses of access points excluded from optimization.", types.StringType),
			"high_priority_devices": optList(
				"MAC addresses of access points optimized first.", types.StringType),
			"channels_blacklist": schema.ListNestedAttribute{
				MarkdownDescription: "Channels Radio AI must not use.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"radio": schema.StringAttribute{
						MarkdownDescription: "Band: `ng`, `na` or `6e`.", Required: true,
						Validators: []validator.String{stringvalidator.OneOf("ng", "na", "6e")},
					},
					"channel":       schema.Int64Attribute{MarkdownDescription: "Channel number.", Required: true},
					"channel_width": schema.Int64Attribute{MarkdownDescription: "Channel width in MHz.", Required: true},
				}},
			},
			"radios_configuration": schema.ListNestedAttribute{
				MarkdownDescription: "Per-band channel width and DFS use.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"radio": schema.StringAttribute{
						MarkdownDescription: "Band: `ng`, `na` or `6e`.", Required: true,
						Validators: []validator.String{stringvalidator.OneOf("ng", "na", "6e")},
					},
					"channel_width": schema.Int64Attribute{MarkdownDescription: "Channel width in MHz.", Required: true},
					"dfs":           schema.BoolAttribute{MarkdownDescription: "Whether DFS channels may be used.", Required: true},
				}},
			},
		},
	}
}

func (r *settingResource) radioAiModelToSetting(
	ctx context.Context,
	m *settingRadioAiModel,
	base *settings.RadioAi,
	diags *diag.Diagnostics,
) *settings.RadioAi {
	s := base
	if isKnown(m.Enabled) {
		s.Enabled = m.Enabled.ValueBool()
	}
	if isKnown(m.AutoEnabled) {
		v := m.AutoEnabled.ValueBool()
		s.AutoEnabled = &v
	}
	if isKnown(m.CronExpr) {
		s.CronExpr = m.CronExpr.ValueString()
	}
	if isKnown(m.SettingPreference) {
		s.SettingPreference = m.SettingPreference.ValueString()
	}
	if isKnown(m.AutoChannelPresetsType) {
		s.AutoChannelPresetsType = m.AutoChannelPresetsType.ValueString()
	}
	if isKnown(m.AutoAdjustChannelsToCountry) {
		s.AutoAdjustChannelsToCountry = m.AutoAdjustChannelsToCountry.ValueBool()
	}
	strs := func(l types.List, dst *[]string) {
		if isKnown(l) {
			var v []string
			diags.Append(l.ElementsAs(ctx, &v, false)...)
			*dst = v
		}
	}
	ints := func(l types.List, dst *[]int64) {
		if isKnown(l) {
			var v []int64
			diags.Append(l.ElementsAs(ctx, &v, false)...)
			*dst = v
		}
	}
	strs(m.Optimize, &s.Optimize)
	strs(m.Radios, &s.Radios)
	ints(m.ChannelsNg, &s.ChannelsNg)
	ints(m.ChannelsNa, &s.ChannelsNa)
	ints(m.Channels6e, &s.Channels6E)
	ints(m.HtModesNg, &s.HtModesNg)
	ints(m.HtModesNa, &s.HtModesNa)
	strs(m.ExcludeDevices, &s.ExcludeDevices)
	strs(m.HighPriorityDevices, &s.HighPriorityDevices)

	if isKnown(m.ChannelsBlacklist) {
		var bl []settingRadioAiBlacklistModel
		diags.Append(m.ChannelsBlacklist.ElementsAs(ctx, &bl, false)...)
		s.ChannelsBlacklist = make([]settings.SettingRadioAiChannelsBlacklist, len(bl))
		for i, b := range bl {
			s.ChannelsBlacklist[i] = settings.SettingRadioAiChannelsBlacklist{
				Radio:        b.Radio.ValueString(),
				Channel:      b.Channel.ValueInt64Pointer(),
				ChannelWidth: b.ChannelWidth.ValueInt64Pointer(),
			}
		}
	}
	if isKnown(m.RadiosConfiguration) {
		var rc []settingRadioAiRadioConfigModel
		diags.Append(m.RadiosConfiguration.ElementsAs(ctx, &rc, false)...)
		s.RadiosConfiguration = make([]settings.SettingRadioAiRadiosConfiguration, len(rc))
		for i, c := range rc {
			s.RadiosConfiguration[i] = settings.SettingRadioAiRadiosConfiguration{
				Radio:        c.Radio.ValueString(),
				ChannelWidth: c.ChannelWidth.ValueInt64Pointer(),
				Dfs:          c.Dfs.ValueBool(),
			}
		}
	}
	return s
}

func (r *settingResource) radioAiSettingToModel(
	ctx context.Context,
	s *settings.RadioAi,
	diags *diag.Diagnostics,
) settingRadioAiModel {
	m := settingRadioAiModel{
		Enabled:                     types.BoolValue(s.Enabled),
		AutoEnabled:                 types.BoolPointerValue(s.AutoEnabled),
		CronExpr:                    types.StringValue(s.CronExpr),
		SettingPreference:           types.StringValue(s.SettingPreference),
		AutoChannelPresetsType:      types.StringValue(s.AutoChannelPresetsType),
		AutoAdjustChannelsToCountry: types.BoolValue(s.AutoAdjustChannelsToCountry),
	}
	if s.AutoEnabled == nil {
		m.AutoEnabled = types.BoolValue(false)
	}
	strList := func(v []string) types.List {
		l, d := types.ListValueFrom(ctx, types.StringType, nonNilStrings(v))
		diags.Append(d...)
		return l
	}
	intList := func(v []int64) types.List {
		if v == nil {
			v = []int64{}
		}
		l, d := types.ListValueFrom(ctx, types.Int64Type, v)
		diags.Append(d...)
		return l
	}
	m.Optimize = strList(s.Optimize)
	m.Radios = strList(s.Radios)
	m.ChannelsNg = intList(s.ChannelsNg)
	m.ChannelsNa = intList(s.ChannelsNa)
	m.Channels6e = intList(s.Channels6E)
	m.HtModesNg = intList(s.HtModesNg)
	m.HtModesNa = intList(s.HtModesNa)
	m.ExcludeDevices = strList(s.ExcludeDevices)
	m.HighPriorityDevices = strList(s.HighPriorityDevices)

	bl := make([]settingRadioAiBlacklistModel, len(s.ChannelsBlacklist))
	for i, b := range s.ChannelsBlacklist {
		bl[i] = settingRadioAiBlacklistModel{
			Radio:        types.StringValue(b.Radio),
			Channel:      types.Int64PointerValue(b.Channel),
			ChannelWidth: types.Int64PointerValue(b.ChannelWidth),
		}
	}
	var d diag.Diagnostics
	m.ChannelsBlacklist, d = types.ListValueFrom(
		ctx, types.ObjectType{AttrTypes: radioAiBlacklistAttrTypes}, bl)
	diags.Append(d...)

	rc := make([]settingRadioAiRadioConfigModel, len(s.RadiosConfiguration))
	for i, c := range s.RadiosConfiguration {
		rc[i] = settingRadioAiRadioConfigModel{
			Radio:        types.StringValue(c.Radio),
			ChannelWidth: types.Int64PointerValue(c.ChannelWidth),
			Dfs:          types.BoolValue(c.Dfs),
		}
	}
	m.RadiosConfiguration, d = types.ListValueFrom(
		ctx, types.ObjectType{AttrTypes: radioAiRadioConfigAttrTypes}, rc)
	diags.Append(d...)
	return m
}

// --- read / write --------------------------------------------------------------

// persistMdnsRadioAi writes the mdns and radio_ai blocks present in the plan.
func (r *settingResource) persistMdnsRadioAi(
	ctx context.Context,
	site string,
	plan *settingResourceModel,
	diags *diag.Diagnostics,
) {
	if isKnown(plan.Mdns) {
		var m settingMdnsModel
		diags.Append(plan.Mdns.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		base, ok := getSettingOrEmpty[*settings.Mdns](ctx, r, site, &settings.Mdns{}, "mDNS", diags)
		if !ok {
			return
		}
		s := r.mdnsModelToSetting(ctx, &m, base, diags)
		if diags.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, s); err != nil {
			diags.AddError("Error Updating mDNS Setting", err.Error())
			return
		}
	}
	if isKnown(plan.RadioAi) {
		var m settingRadioAiModel
		diags.Append(plan.RadioAi.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		base, ok := getSettingOrEmpty[*settings.RadioAi](ctx, r, site, &settings.RadioAi{}, "Radio AI", diags)
		if !ok {
			return
		}
		s := r.radioAiModelToSetting(ctx, &m, base, diags)
		if diags.HasError() {
			return
		}
		if err := r.client.UpdateSetting(ctx, site, s); err != nil {
			diags.AddError("Error Updating Radio AI Setting", err.Error())
			return
		}
	}
}

// readMdnsRadioAi refreshes the mdns and radio_ai blocks that are present in data.
func (r *settingResource) readMdnsRadioAi(
	ctx context.Context,
	site string,
	data *settingResourceModel,
	diags *diag.Diagnostics,
) {
	if isKnown(data.Mdns) {
		_, s, err := ui.GetSetting[*settings.Mdns](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading mDNS Setting", err.Error())
			return
		}
		obj, d := types.ObjectValueFrom(ctx, mdnsAttrTypes, r.mdnsSettingToModel(ctx, s, diags))
		diags.Append(d...)
		data.Mdns = keepNullAttributes(data.Mdns, obj, diags)
	} else {
		data.Mdns = types.ObjectNull(mdnsAttrTypes)
	}
	if isKnown(data.RadioAi) {
		_, s, err := ui.GetSetting[*settings.RadioAi](r.client.ApiClient, ctx, site)
		if err != nil {
			diags.AddError("Error Reading Radio AI Setting", err.Error())
			return
		}
		obj, d := types.ObjectValueFrom(ctx, radioAiAttrTypes, r.radioAiSettingToModel(ctx, s, diags))
		diags.Append(d...)
		data.RadioAi = keepNullAttributes(data.RadioAi, obj, diags)
	} else {
		data.RadioAi = types.ObjectNull(radioAiAttrTypes)
	}
}

// getSettingOrEmpty fetches a setting as the base for a read-modify-write; a
// setting the controller has never stored yields empty.
func getSettingOrEmpty[T settings.Setting](
	ctx context.Context,
	r *settingResource,
	site string,
	empty T,
	label string,
	diags *diag.Diagnostics,
) (T, bool) {
	_, s, err := ui.GetSetting[T](r.client.ApiClient, ctx, site)
	if err != nil {
		var notFound *ui.NotFoundError
		if !errors.As(err, &notFound) {
			diags.AddError("Error Reading "+label+" Setting", err.Error())
			return empty, false
		}
		return empty, true
	}
	return s, true
}

// keepNullAttributes returns fresh, except that attributes null in prior stay
// null. An attribute left out of config is planned as null after an import
// (UseStateForUnknown copies the empty prior), so filling it from the
// controller on read-back would be an inconsistent result. Like the other
// blocks, such attributes are left to the controller and not tracked.
func keepNullAttributes(prior, fresh types.Object, diags *diag.Diagnostics) types.Object {
	if prior.IsNull() || prior.IsUnknown() || fresh.IsNull() {
		return fresh
	}
	attrs := fresh.Attributes()
	for k, v := range prior.Attributes() {
		if v.IsNull() {
			attrs[k] = v
		}
	}
	obj, d := types.ObjectValue(fresh.AttributeTypes(context.Background()), attrs)
	diags.Append(d...)
	return obj
}

func isKnown(v attr.Value) bool { return !v.IsNull() && !v.IsUnknown() }

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
