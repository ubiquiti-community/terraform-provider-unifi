package unifi

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// unifi_device port_override forms of the shared port settings (port_settings.go).

// devicePortQOSProfileModes are the qos_profile_mode values a switch port
// override takes: the port profile's, plus `smpte_st2110`.
var devicePortQOSProfileModes = append(append([]string{}, portQOSProfileModes...), "smpte_st2110")

// devicePortLinkDebounceSchema and devicePortQOSProfileSchema are the
// unifi_device port_override forms. A port_override is an element of a set, so
// they carry no plan modifiers (there is no prior element to keep). qos_profile
// is only managed when set there: an unset one keeps the port's profile, so the
// object is Optional (not Computed) and its leaves resolve from the controller.
func devicePortLinkDebounceSchema() schema.SingleNestedAttribute {
	a := portLinkDebounceSchema("port")
	a.PlanModifiers = nil
	return a
}

func devicePortQOSProfileSchema() schema.SingleNestedAttribute {
	a := portQOSProfileSchema("port", devicePortQOSProfileModes)
	a.Computed = false
	a.PlanModifiers = nil
	a.Description = "QoS profile for the port (the UI's Pro AV and custom QoS): a preset `mode` for an AV " +
		"protocol, or `custom` with your own `policies`. Only managed when set: leaving it unset keeps the " +
		"port's current profile, and `qos_profile = {}` removes it."
	return a
}

// deviceQOSToAPI and deviceQOSFromAPI convert between the neutral form and
// go-unifi's DeviceQOSProfile (a switch port override).
func deviceQOSToAPI(q *portQOSProfileAPI) *unifi.DeviceQOSProfile {
	if q == nil {
		return nil
	}
	out := &unifi.DeviceQOSProfile{QOSProfileMode: q.Mode}
	for _, p := range q.Policies {
		var pp unifi.DeviceQOSPolicies
		if m := p.Matching; m != nil {
			pp.QOSMatching = &unifi.DeviceQOSMatching{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				DstPort:          m.DstPort,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Protocol:         m.Protocol,
				SrcPort:          m.SrcPort,
			}
		}
		if m := p.Marking; m != nil {
			pp.QOSMarking = &unifi.DeviceQOSMarking{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Queue:            m.Queue,
			}
		}
		out.QOSPolicies = append(out.QOSPolicies, pp)
	}
	return out
}

func deviceQOSFromAPI(q *unifi.DeviceQOSProfile) *portQOSProfileAPI {
	if q == nil {
		return nil
	}
	out := &portQOSProfileAPI{Mode: q.QOSProfileMode}
	for _, p := range q.QOSPolicies {
		var np portQOSPolicyAPI
		if m := p.QOSMatching; m != nil {
			np.Matching = &portQOSMatchingAPI{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				DstPort:          m.DstPort,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Protocol:         m.Protocol,
				SrcPort:          m.SrcPort,
			}
		}
		if m := p.QOSMarking; m != nil {
			np.Marking = &portQOSMarkingAPI{
				CosCode:          m.CosCode,
				DscpCode:         m.DscpCode,
				IPPrecedenceCode: m.IPPrecedenceCode,
				Queue:            m.Queue,
			}
		}
		out.Policies = append(out.Policies, np)
	}
	return out
}
