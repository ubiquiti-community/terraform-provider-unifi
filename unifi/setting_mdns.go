package unifi

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ui "github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// The mdns block of unifi_setting. It writes by read-modify-write:
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
				MarkdownDescription: "Networks mDNS is enabled for: `all`, `some` (the networks " +
					"with `multicast_dns` enabled on `unifi_network`), or `none`.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("all", "some", "none")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enabled_for_network_ids": schema.ListAttribute{
				MarkdownDescription: "IDs of the networks mDNS is enabled for when `enabled_for` is " +
					"`some`. Read-only: the controller derives this list from each network's " +
					"`mdns_enabled` flag and ignores writes to it, so choose the networks with " +
					"`multicast_dns` on `unifi_network`.",
				ElementType: types.StringType,
				Computed:    true,
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
	// enabled_for_network_ids is derived from the networks' mdns_enabled flags;
	// the stored value in base is sent back unchanged.
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
	m.EnabledForNetworkIDs, d = types.ListValueFrom(
		ctx,
		types.StringType,
		nonNilStrings(s.EnabledForNetworkIDs),
	)
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

// --- read / write --------------------------------------------------------------

// persistMdns writes the mdns block if it is present in the plan.
func (r *settingResource) persistMdns(
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
}

// readMdns refreshes the mdns block if it is present in data.
func (r *settingResource) readMdns(
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
