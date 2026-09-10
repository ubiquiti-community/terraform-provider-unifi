package unifi

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// This file holds the nested-object models shared by unifi_device.port_override
// and unifi_port_profile. Both expose the same per-port feature groups, which
// were previously flat prefixed attributes (dot1x_ctrl, stormctrl_bcast_level,
// …) and are now bundled:
//
//	dot1x             = { ctrl, idle_timeout }
//	egress_rate_limit = { enabled, kbps }
//	lldpmed           = { enabled, notify_enabled }
//	port_security     = { enabled, mac_address }
//	stormctrl         = { type, bcast = {…}, mcast = {…}, ucast = {…} }
//
// The models, attribute types, API<->object builders and the state-upgrade
// rewriter live here so the two resources cannot drift apart.

// portDot1xModel is the `dot1x` nested object.
type portDot1xModel struct {
	Ctrl        types.String         `tfsdk:"ctrl"`
	IdleTimeout timetypes.GoDuration `tfsdk:"idle_timeout"`
}

func portDot1xAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ctrl":         types.StringType,
		"idle_timeout": timetypes.GoDurationType{},
	}
}

// portEgressRateLimitModel is the `egress_rate_limit` nested object.
type portEgressRateLimitModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Kbps    types.Int64 `tfsdk:"kbps"`
}

func portEgressRateLimitAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"kbps":    types.Int64Type,
	}
}

// portLldpmedModel is the `lldpmed` nested object.
type portLldpmedModel struct {
	Enabled       types.Bool `tfsdk:"enabled"`
	NotifyEnabled types.Bool `tfsdk:"notify_enabled"`
}

func portLldpmedAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled":        types.BoolType,
		"notify_enabled": types.BoolType,
	}
}

// portStormctrlClassModel is one traffic class (`bcast`, `mcast`, `ucast`)
// inside the `stormctrl` nested object.
type portStormctrlClassModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Level   types.Int64 `tfsdk:"level"`
	Rate    types.Int64 `tfsdk:"rate"`
}

func portStormctrlClassAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"level":   types.Int64Type,
		"rate":    types.Int64Type,
	}
}

// portStormctrlModel is the `stormctrl` nested object.
type portStormctrlModel struct {
	Type  types.String `tfsdk:"type"`
	Bcast types.Object `tfsdk:"bcast"`
	Mcast types.Object `tfsdk:"mcast"`
	Ucast types.Object `tfsdk:"ucast"`
}

func portStormctrlAttrTypes() map[string]attr.Type {
	class := types.ObjectType{AttrTypes: portStormctrlClassAttrTypes()}
	return map[string]attr.Type{
		"type":  types.StringType,
		"bcast": class,
		"mcast": class,
		"ucast": class,
	}
}

// portStormctrlAPI is the flat storm-control field set shared by
// unifi.DevicePortOverrides and unifi.PortProfile, gathered so one builder can
// produce the nested object for both resources.
type portStormctrlAPI struct {
	Type         string
	BcastEnabled bool
	BcastLevel   *int64
	BcastRate    *int64
	McastEnabled bool
	McastLevel   *int64
	McastRate    *int64
	UcastEnabled bool
	UcastLevel   *int64
	UcastRate    *int64
}

func portStormctrlClassObject(
	ctx context.Context,
	enabled bool,
	level, rate *int64,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, portStormctrlClassAttrTypes(), portStormctrlClassModel{
		Enabled: types.BoolValue(enabled),
		Level:   types.Int64PointerValue(level),
		Rate:    types.Int64PointerValue(rate),
	})
}

// portStormctrlObject builds the `stormctrl` object from API values.
func portStormctrlObject(
	ctx context.Context,
	api portStormctrlAPI,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	bcast, d := portStormctrlClassObject(ctx, api.BcastEnabled, api.BcastLevel, api.BcastRate)
	diags.Append(d...)
	mcast, d := portStormctrlClassObject(ctx, api.McastEnabled, api.McastLevel, api.McastRate)
	diags.Append(d...)
	ucast, d := portStormctrlClassObject(ctx, api.UcastEnabled, api.UcastLevel, api.UcastRate)
	diags.Append(d...)
	if diags.HasError() {
		return types.ObjectNull(portStormctrlAttrTypes()), diags
	}
	obj, d := types.ObjectValueFrom(ctx, portStormctrlAttrTypes(), portStormctrlModel{
		Type:  util.StringValueOrNull(api.Type),
		Bcast: bcast,
		Mcast: mcast,
		Ucast: ucast,
	})
	diags.Append(d...)
	return obj, diags
}

// portStormctrlFromObject decodes a `stormctrl` object back into flat API
// values. Null/unknown objects (or classes) contribute nothing, leaving the
// returned struct at its zero value, exactly as the flat attributes did.
func portStormctrlFromObject(
	ctx context.Context,
	obj types.Object,
) (portStormctrlAPI, diag.Diagnostics) {
	var out portStormctrlAPI
	m, ok, diags := util.ObjectAs[portStormctrlModel](ctx, obj)
	if !ok {
		return out, diags
	}
	if !m.Type.IsNull() && !m.Type.IsUnknown() {
		out.Type = m.Type.ValueString()
	}
	class := func(o types.Object, enabled *bool, level, rate **int64) {
		c, ok, d := util.ObjectAs[portStormctrlClassModel](ctx, o)
		diags.Append(d...)
		if !ok {
			return
		}
		*enabled = c.Enabled.ValueBool()
		if !c.Level.IsNull() && !c.Level.IsUnknown() {
			*level = c.Level.ValueInt64Pointer()
		}
		if !c.Rate.IsNull() && !c.Rate.IsUnknown() {
			*rate = c.Rate.ValueInt64Pointer()
		}
	}
	class(m.Bcast, &out.BcastEnabled, &out.BcastLevel, &out.BcastRate)
	class(m.Mcast, &out.McastEnabled, &out.McastLevel, &out.McastRate)
	class(m.Ucast, &out.UcastEnabled, &out.UcastLevel, &out.UcastRate)
	return out, diags
}

// portDot1xObject builds the `dot1x` object from API values. ctrl "" maps to
// null; a nil idle timeout maps to a null duration.
func portDot1xObject(
	ctx context.Context,
	ctrl string,
	idleTimeout *int64,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, portDot1xAttrTypes(), portDot1xModel{
		Ctrl:        util.StringValueOrNull(ctrl),
		IdleTimeout: util.DurationPtrValue(idleTimeout, time.Second),
	})
}

// portEgressRateLimitObject builds the `egress_rate_limit` object from API values.
func portEgressRateLimitObject(
	ctx context.Context,
	enabled bool,
	kbps *int64,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, portEgressRateLimitAttrTypes(), portEgressRateLimitModel{
		Enabled: types.BoolValue(enabled),
		Kbps:    types.Int64PointerValue(kbps),
	})
}

// portLldpmedObject builds the `lldpmed` object from API values.
func portLldpmedObject(
	ctx context.Context,
	enabled, notifyEnabled bool,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, portLldpmedAttrTypes(), portLldpmedModel{
		Enabled:       types.BoolValue(enabled),
		NotifyEnabled: types.BoolValue(notifyEnabled),
	})
}

// nestPortGroupState rewrites one flat port record (a port_override element or
// a port profile) in decoded prior state into the nested-object layout. It is
// shared by the unifi_device and unifi_port_profile state upgraders.
func nestPortGroupState(po map[string]any) {
	util.NestFields(po, "dot1x", map[string]string{
		"dot1x_ctrl":         "ctrl",
		"dot1x_idle_timeout": "idle_timeout",
	})
	util.NestFields(po, "egress_rate_limit", map[string]string{
		"egress_rate_limit_kbps_enabled": "enabled",
		"egress_rate_limit_kbps":         "kbps",
	})
	util.NestFields(po, "lldpmed", map[string]string{
		"lldpmed_enabled":        "enabled",
		"lldpmed_notify_enabled": "notify_enabled",
	})
	util.NestFields(po, "port_security", map[string]string{
		"port_security_enabled":     "enabled",
		"port_security_mac_address": "mac_address",
	})
	util.NestFields(po, "stormctrl", map[string]string{
		"stormctrl_type":          "type",
		"stormctrl_bcast_enabled": "bcast_enabled",
		"stormctrl_bcast_level":   "bcast_level",
		"stormctrl_bcast_rate":    "bcast_rate",
		"stormctrl_mcast_enabled": "mcast_enabled",
		"stormctrl_mcast_level":   "mcast_level",
		"stormctrl_mcast_rate":    "mcast_rate",
		"stormctrl_ucast_enabled": "ucast_enabled",
		"stormctrl_ucast_level":   "ucast_level",
		"stormctrl_ucast_rate":    "ucast_rate",
	})
	util.WithObject(po, "stormctrl", func(sc map[string]any) {
		for _, class := range []string{"bcast", "mcast", "ucast"} {
			util.NestFields(sc, class, map[string]string{
				class + "_enabled": "enabled",
				class + "_level":   "level",
				class + "_rate":    "rate",
			})
		}
	})
}
