package unifi

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/planmodifiers"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &vpnServerResource{}
	_ resource.ResourceWithImportState  = &vpnServerResource{}
	_ resource.ResourceWithIdentity     = &vpnServerResource{}
	_ resource.ResourceWithUpgradeState = &vpnServerResource{}
)

// Ensure provider defined types fully satisfy list interfaces.
var (
	_ list.ListResource              = &vpnServerResource{}
	_ list.ListResourceWithConfigure = &vpnServerResource{}
)

func NewVPNServerResource() resource.Resource {
	return &vpnServerResource{}
}

func NewVPNServerListResource() list.ListResource {
	return &vpnServerResource{}
}

// vpnServerResource defines the resource implementation.
type vpnServerResource struct {
	client *Client
}

type vpnServerIdentityModel struct {
	ID   types.String `tfsdk:"id"`
	Site types.String `tfsdk:"site"`
}

// vpnServerListConfigModel describes the list configuration model.
type vpnServerListConfigModel struct {
	Site   types.String `tfsdk:"site"`
	Filter types.List   `tfsdk:"filter"`
}

// vpnServerListFilterModel represents a single name/value filter entry.
type vpnServerListFilterModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// vpnServerDNSModel describes the DNS configuration for VPN clients.
type vpnServerDNSModel struct {
	Enabled types.Bool `tfsdk:"enabled"`
	Servers types.List `tfsdk:"servers"`
}

func (m vpnServerDNSModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"enabled": types.BoolType,
		"servers": types.ListType{ElemType: types.StringType},
	}
}

// vpnServerWANModel describes the WAN binding configuration shared across VPN types.
type vpnServerWANModel struct {
	IP        types.String `tfsdk:"ip"`
	Interface types.String `tfsdk:"interface"`
}

func (m vpnServerWANModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"ip":        types.StringType,
		"interface": types.StringType,
	}
}

// vpnServerWireguardModel describes the WireGuard-specific server configuration.
type vpnServerWireguardModel struct {
	PrivateKey          types.String `tfsdk:"private_key"`
	PrivateKeyWO        types.String `tfsdk:"private_key_wo"`
	PrivateKeyWOVersion types.Int64  `tfsdk:"private_key_wo_version"`
	PublicKey           types.String `tfsdk:"public_key"`
	Port                types.Int64  `tfsdk:"port"`
	IPVersion           types.String `tfsdk:"ip_version"`
}

func (m vpnServerWireguardModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"private_key":            types.StringType,
		"private_key_wo":         types.StringType,
		"private_key_wo_version": types.Int64Type,
		"public_key":             types.StringType,
		"port":                   types.Int64Type,
		"ip_version":             types.StringType,
	}
}

// vpnServerL2TPModel describes the L2TP-specific server configuration.
type vpnServerL2TPModel struct {
	AllowWeakCiphers types.Bool   `tfsdk:"allow_weak_ciphers"`
	PreSharedKey     types.String `tfsdk:"pre_shared_key"`
}

func (m vpnServerL2TPModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"allow_weak_ciphers": types.BoolType,
		"pre_shared_key":     types.StringType,
	}
}

// vpnServerCertPairModel is one controller-generated certificate/private key
// pair inside the OpenVPN configuration (`ca`, `server`, `shared_client`).
type vpnServerCertPairModel struct {
	Crt types.String `tfsdk:"crt"`
	Key types.String `tfsdk:"key"`
}

func (m vpnServerCertPairModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"crt": types.StringType,
		"key": types.StringType,
	}
}

// vpnServerOpenVPNModel describes the OpenVPN-specific server configuration.
type vpnServerOpenVPNModel struct {
	Port             types.Int64  `tfsdk:"port"`
	Mode             types.String `tfsdk:"mode"`
	EncryptionCipher types.String `tfsdk:"encryption_cipher"`
	Server           types.Object `tfsdk:"server"`
	DhKey            types.String `tfsdk:"dh_key"`
	SharedClient     types.Object `tfsdk:"shared_client"`
	AuthKey          types.String `tfsdk:"auth_key"`
	Ca               types.Object `tfsdk:"ca"`
}

func (m vpnServerOpenVPNModel) AttributeTypes() map[string]attr.Type {
	certPair := types.ObjectType{AttrTypes: vpnServerCertPairModel{}.AttributeTypes()}
	return map[string]attr.Type{
		"port":              types.Int64Type,
		"mode":              types.StringType,
		"encryption_cipher": types.StringType,
		"server":            certPair,
		"dh_key":            types.StringType,
		"shared_client":     certPair,
		"auth_key":          types.StringType,
		"ca":                certPair,
	}
}

// vpnServerCertPairPointers unpacks a `crt`/`key` object into the API's
// pointer fields with exactly the wire semantics of the flat leaves it
// replaced: a null object (or leaf) is a nil pointer and is omitted, while an
// unknown object (create, before the controller has generated the material)
// sends "" for both leaves just as the unknown flat leaves did via
// ValueStringPointer, which the controller accepts and overwrites.
func vpnServerCertPairPointers(
	ctx context.Context,
	obj types.Object,
) (*string, *string, diag.Diagnostics) {
	if obj.IsUnknown() {
		return new(""), new(""), nil
	}
	pair, ok, diags := util.ObjectAs[vpnServerCertPairModel](ctx, obj)
	if !ok {
		return nil, nil, diags
	}
	return pair.Crt.ValueStringPointer(), pair.Key.ValueStringPointer(), diags
}

// vpnServerCertPairObject builds a `crt`/`key` object from the API's pointer
// fields; a nil pointer becomes a null leaf.
func vpnServerCertPairObject(
	ctx context.Context,
	crt, key *string,
) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(
		ctx,
		vpnServerCertPairModel{}.AttributeTypes(),
		vpnServerCertPairModel{
			Crt: types.StringPointerValue(crt),
			Key: types.StringPointerValue(key),
		},
	)
}

// vpnServerResourceModel describes the resource data model.
type vpnServerResourceModel struct {
	ID              types.String         `tfsdk:"id"`
	Site            types.String         `tfsdk:"site"`
	Name            types.String         `tfsdk:"name"`
	Enabled         types.Bool           `tfsdk:"enabled"`
	Subnet          cidrtypes.IPv4Prefix `tfsdk:"subnet"`
	SubnetIPv6      cidrtypes.IPv6Prefix `tfsdk:"subnet_ipv6"`
	DNS             types.Object         `tfsdk:"dns"`
	WAN             types.Object         `tfsdk:"wan"`
	RADIUSProfileID types.String         `tfsdk:"radiusprofile_id"`
	Wireguard       types.Object         `tfsdk:"wireguard"`
	L2TP            types.Object         `tfsdk:"l2tp"`
	OpenVPN         types.Object         `tfsdk:"openvpn"`
	Timeouts        timeouts.Value       `tfsdk:"timeouts"`
}

func (r *vpnServerResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_vpn_server"
}

// IdentitySchema implements [resource.ResourceWithIdentity].
func (r *vpnServerResource) IdentitySchema(
	_ context.Context,
	_ resource.IdentitySchemaRequest,
	resp *resource.IdentitySchemaResponse,
) {
	resp.IdentitySchema = identityschema.Schema{
		Version: 2,
		// The optional "site" attribute defaults to the provider site on
		// import. Identities stored by older provider versions ({id} only)
		// are version 0 and go through siteIdentityUpgraders, which fills
		// site from the provider's configured site.
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"site": identityschema.StringAttribute{
				OptionalForImport: true,
			},
		},
	}
}

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity]. See
// siteIdentityUpgraders.
func (r *vpnServerResource) UpgradeIdentity(
	_ context.Context,
) map[int64]resource.IdentityUpgrader {
	return siteIdentityUpgraders(func() *Client { return r.client })
}

func (r *vpnServerResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	wanInterfaceValidator := stringvalidator.RegexMatches(
		regexp.MustCompile(`^wan[2-9]?$`),
		"must be 'wan' or 'wan2' through 'wan9'",
	)

	resp.Schema = schema.Schema{
		MarkdownDescription: "`unifi_vpn_server` manages VPN server configurations (WireGuard, L2TP, or OpenVPN) in the UniFi controller.",
		// Version history:
		//   0: initial schema (flat openvpn.{ca,server,shared_client}_{crt,key}).
		//   1: the certificate/key pairs are nested objects. See UpgradeState.
		Version: 1,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the VPN server.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"site": schema.StringAttribute{
				MarkdownDescription: "The name of the site to associate the VPN server with.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the VPN server.",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Specifies whether the VPN server is enabled. A new server " +
					"is created enabled; an existing one keeps the controller's value when " +
					"this is not set, so adopting a disabled server does not re-enable it (#544).",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					planmodifiers.BoolDefaultOnCreate(true),
				},
			},
			"subnet": schema.StringAttribute{
				MarkdownDescription: "The VPN server subnet in CIDR notation (e.g., `10.100.0.1/24`). " +
					"The first address is the server's tunnel IP, so use the gateway form " +
					"(`10.100.0.1/24`) — the network address (`10.100.0.0/24`) is rejected with " +
					"`api.err.IncorrectIPSubnetSpec`.",
				Required:   true,
				CustomType: cidrtypes.IPv4PrefixType{},
			},
			"subnet_ipv6": schema.StringAttribute{
				MarkdownDescription: "The VPN server's IPv6 subnet in CIDR notation, in the same gateway " +
					"form as `subnet` (e.g., `fd00:db8::1/64`): the UI's \"IPv6 Gateway/Subnet\". " +
					"Clients get an address in it (`unifi_wireguard_peer.interface_ipv6`). " +
					"The controller only accepts it when a server is created, so changing " +
					"it replaces the server (and with it the server's key).",
				Optional:   true,
				CustomType: cidrtypes.IPv6PrefixType{},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dns": schema.SingleNestedAttribute{
				MarkdownDescription: "DNS configuration pushed to VPN clients.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						MarkdownDescription: "Specifies whether custom DNS servers are enabled for VPN clients. Defaults to `true` when `servers` is non-empty.",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"servers": schema.ListAttribute{
						MarkdownDescription: "DNS servers to push to VPN clients.",
						Optional:            true,
						ElementType:         types.StringType,
					},
				},
			},
			"wan": schema.SingleNestedAttribute{
				MarkdownDescription: "WAN binding configuration for the VPN server.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "Local WAN IP to bind the VPN server to. Use `any` to " +
							"listen on all addresses. Defaults to `any` when the `wan` block is " +
							"declared on a new server; an existing server keeps the controller's " +
							"value when this is not set, including no value at all, which is how " +
							"a server created in the UI is stored (#544).",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							planmodifiers.StringDefaultOnCreateKeepNull("any"),
						},
					},
					"interface": schema.StringAttribute{
						MarkdownDescription: "WAN interface to use for the VPN server (e.g., `wan`, " +
							"`wan2`). Defaults to `wan` when the `wan` block is declared on a new " +
							"server; an existing server keeps the controller's value when this is " +
							"not set, so adopting a server bound to `wan2` does not move it (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.String{
							wanInterfaceValidator,
						},
						PlanModifiers: []planmodifier.String{
							planmodifiers.StringDefaultOnCreateKeepNull("wan"),
						},
					},
				},
			},
			"radiusprofile_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the RADIUS profile to use for authentication. Applicable to L2TP and OpenVPN server types.",
				Optional:            true,
			},
			"wireguard": schema.SingleNestedAttribute{
				MarkdownDescription: "WireGuard VPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"private_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard private key for this server (base64). If neither " +
							"this nor `private_key_wo` is specified, the provider generates one at " +
							"create time (the controller does not generate it and rejects a server " +
							"without a key). Stored in state; use `private_key_wo` to avoid " +
							"persisting the secret.",
						Optional:  true,
						Computed:  true,
						Sensitive: true,
						Validators: []validator.String{
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("private_key_wo"),
							),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							vpnServerPrivateKeyNullWithWO{},
						},
					},
					"private_key_wo": schema.StringAttribute{
						MarkdownDescription: "Write-only equivalent of `private_key` (Terraform 1.11+). " +
							"Used at apply time but never written to state, so it can be sourced " +
							"from an ephemeral resource; `private_key` then stays null in state too. " +
							"The key is sent on create and whenever `private_key_wo_version` " +
							"changes; other updates keep the key the controller holds. Mutually " +
							"exclusive with `private_key`.",
						Optional:  true,
						Sensitive: true,
						WriteOnly: true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
							stringvalidator.ConflictsWith(
								path.MatchRelative().AtParent().AtName("private_key"),
							),
						},
					},
					"private_key_wo_version": schema.Int64Attribute{
						MarkdownDescription: "Version counter for `private_key_wo`. Increment this " +
							"value to send a new `private_key_wo` to the controller (key rotation).",
						Optional: true,
						Validators: []validator.Int64{
							int64validator.AtLeast(1),
							int64validator.AlsoRequires(
								path.MatchRelative().AtParent().AtName("private_key_wo"),
							),
						},
					},
					"public_key": schema.StringAttribute{
						MarkdownDescription: "WireGuard public key for this server. Computed from the private key.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							vpnServerPublicKeyUnknownOnKeyChange{},
						},
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "UDP port for the WireGuard server to listen on. " +
							"Defaults to `51820` for a new server; an existing server keeps the " +
							"controller's port when this is not set (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"ip_version": schema.StringAttribute{
						MarkdownDescription: "The address family the server listens on, `v4` or `v6`: " +
							"one per server (the UI's IP Address choice between the WAN's IPv4 " +
							"and IPv6 addresses). Defaults to `v4` for a new server, which is also " +
							"what a server without the setting does; an existing server keeps the " +
							"controller's value when this is not set (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.String{
							stringvalidator.OneOf("v4", "v6"),
						},
						PlanModifiers: []planmodifier.String{
							planmodifiers.StringDefaultOnCreate("v4"),
						},
					},
				},
			},
			"l2tp": schema.SingleNestedAttribute{
				MarkdownDescription: "L2TP VPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"allow_weak_ciphers": schema.BoolAttribute{
						MarkdownDescription: "Allow weak ciphers for L2TP connections. Taken " +
							"from the controller when not set (a new server gets `false`) (#544).",
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"pre_shared_key": schema.StringAttribute{
						MarkdownDescription: "IPsec pre-shared key for L2TP. Required by the UniFi controller.",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
			"openvpn": schema.SingleNestedAttribute{
				MarkdownDescription: "OpenVPN server configuration. Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"port": schema.Int64Attribute{
						MarkdownDescription: "Port for the OpenVPN server to listen on. " +
							"Defaults to `1194` for a new server; an existing server keeps the " +
							"controller's port when this is not set (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"mode": schema.StringAttribute{
						MarkdownDescription: "OpenVPN mode. Defaults to `server` for a new " +
							"server; an existing server keeps the controller's value when this " +
							"is not set (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.String{
							stringvalidator.OneOf("server", "site-to-site"),
						},
						PlanModifiers: []planmodifier.String{
							planmodifiers.StringDefaultOnCreateKeepNull("server"),
						},
					},
					"encryption_cipher": schema.StringAttribute{
						MarkdownDescription: "Encryption cipher for OpenVPN. Defaults to " +
							"`AES_256_GCM` for a new server; an existing server keeps the " +
							"controller's value when this is not set (#544).",
						Optional: true,
						Computed: true,
						Validators: []validator.String{
							stringvalidator.OneOf("AES_256_GCM", "AES_256_CBC", "BF_CBC"),
						},
						PlanModifiers: []planmodifier.String{
							planmodifiers.StringDefaultOnCreateKeepNull("AES_256_GCM"),
						},
					},
					"server": vpnServerCertPairSchema("Server"),
					"dh_key": schema.StringAttribute{
						MarkdownDescription: "Diffie-Hellman parameters generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"shared_client": vpnServerCertPairSchema("Shared client"),
					"auth_key": schema.StringAttribute{
						MarkdownDescription: "OpenVPN static authentication key generated by the controller.",
						Computed:            true,
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"ca": vpnServerCertPairSchema("CA"),
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

// vpnServerCertPairSchema is the schema of one controller-generated
// certificate/private key pair (`ca`, `server`, `shared_client`) inside
// `openvpn`. Both leaves are computed and sensitive, and the object is
// resolved from prior state on plan just like the flat leaves it replaced.
func vpnServerCertPairSchema(what string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: what + " certificate and private key generated by the controller.",
		Computed:            true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"crt": schema.StringAttribute{
				MarkdownDescription: what + " certificate generated by the controller.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: what + " private key generated by the controller.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// UpgradeState migrates prior VPN server state to the current schema version.
//
//	v0 -> v1: the flat openvpn.{ca,server,shared_client}_{crt,key} attributes
//	    move into nested openvpn.{ca,server,shared_client} = { crt, key }
//	    objects. See nestVPNServerState.
func (r *vpnServerResource) UpgradeState(
	ctx context.Context,
) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	upgrader := func(rewrite func(state map[string]any)) resource.StateUpgrader {
		return resource.StateUpgrader{
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				if req.RawState == nil {
					return
				}
				dv, err := util.UpgradeRawState(
					schemaType,
					req.RawState.JSON,
					func(state map[string]any) {
						rewrite(state)
						nestVPNServerState(state)
					},
				)
				if err != nil {
					resp.Diagnostics.AddError("Failed to upgrade VPN server state", err.Error())
					return
				}
				resp.DynamicValue = dv
			},
		}
	}

	return map[int64]resource.StateUpgrader{
		0: upgrader(func(map[string]any) {}),
	}
}

// nestVPNServerState rewrites flat v0 VPN server state into the nested-object
// shape of the current schema: the OpenVPN certificate/key attributes move
// under openvpn.{ca,server,shared_client}.
func nestVPNServerState(state map[string]any) {
	util.WithObject(state, "openvpn", func(openvpn map[string]any) {
		util.NestFields(openvpn, "ca", map[string]string{
			"ca_crt": "crt",
			"ca_key": "key",
		})
		util.NestFields(openvpn, "server", map[string]string{
			"server_crt": "crt",
			"server_key": "key",
		})
		util.NestFields(openvpn, "shared_client", map[string]string{
			"shared_client_crt": "crt",
			"shared_client_key": "key",
		})
	})
}

func (r *vpnServerResource) Configure(
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

func (r *vpnServerResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := data.Timeouts.Create(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	vpnServerCreateDefaults(network)

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	privateKeyWO := r.readPrivateKeyWO(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.resolveWireguardPrivateKey(
		ctx, network, site, "", privateKeyWO, true,
	)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createdNetwork, err := r.client.CreateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating VPN Server",
			err.Error(),
		)
		return
	}

	var planData vpnServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, createdNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	idModel := vpnServerIdentityModel{ID: data.ID, Site: data.Site}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, timeoutDiags := data.Timeouts.Read(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	// Get the resource identity, tolerating null/empty identities (state
	// written by older provider versions).
	var identity vpnServerIdentityModel
	if req.Identity != nil && !req.Identity.Raw.IsFullyNull() {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	site := data.Site.ValueString()
	if site == "" {
		site = identity.Site.ValueString()
	}
	if site == "" {
		site = r.client.Site
	}

	// Prefer the state ID; fall back to the identity ID (the post-import
	// refresh may run from an identity-only state).
	lookupID := ""
	if !data.ID.IsNull() && !data.ID.IsUnknown() {
		lookupID = data.ID.ValueString()
	}
	if lookupID == "" && !identity.ID.IsNull() && !identity.ID.IsUnknown() {
		lookupID = identity.ID.ValueString()
	}

	var err error
	var network *unifi.Network

	if lookupID != "" {
		network, err = r.client.GetNetwork(ctx, site, lookupID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPN Server",
				"Could not read VPN server ID "+lookupID+": "+err.Error(),
			)
			return
		}
	} else {
		network, err = r.client.GetNetworkByName(ctx, site, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VPN Server",
				"Could not read VPN server name "+data.Name.ValueString()+": "+err.Error(),
			)
			return
		}
	}

	var priorState vpnServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags := r.networkToModel(ctx, network, &data, site, &priorState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Terraform rejects any modification of an existing stored identity
	// (including filling a null attribute), so pass an incoming identity
	// through unchanged and only derive one when none is stored yet.
	if resp.Identity != nil && (req.Identity == nil || req.Identity.Raw.IsFullyNull()) {
		idModel := vpnServerIdentityModel{ID: data.ID, Site: data.Site}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := data.Timeouts.Update(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	network, diags := r.modelToNetwork(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	network.ID = data.ID.ValueString()

	privateKeyWO := r.readPrivateKeyWO(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var priorState vpnServerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sendWO, d := vpnServerSendPrivateKeyWO(ctx, &data, &priorState)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !sendWO {
		privateKeyWO = types.StringNull()
	}
	resp.Diagnostics.Append(r.resolveWireguardPrivateKey(
		ctx, network, site, network.ID, privateKeyWO, false,
	)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedNetwork, err := r.client.UpdateNetwork(ctx, site, network)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating VPN Server",
			err.Error(),
		)
		return
	}

	var planData vpnServerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = r.networkToModel(ctx, updatedNetwork, &data, site, &planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pass an existing identity through unchanged; only derive one when
	// none is stored yet (see Read).
	if resp.Identity != nil && (req.Identity == nil || req.Identity.Raw.IsFullyNull()) {
		idModel := vpnServerIdentityModel{ID: data.ID, Site: data.Site}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *vpnServerResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data vpnServerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := data.Timeouts.Delete(ctx, 20*time.Minute)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	site := data.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	name := data.Name.ValueString()
	err := r.client.DeleteNetwork(ctx, site, data.ID.ValueString(), name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting VPN Server",
			err.Error(),
		)
		return
	}
}

func (r *vpnServerResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	// Identity-based import (Terraform 1.12+ import block with identity).
	if req.ID == "" {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Invalid Import Request",
				"Importing a VPN server requires either an import ID or a resource identity.",
			)
			return
		}
		var identity vpnServerIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Invalid Import Identity",
				"The `id` identity attribute is required to import a VPN server.",
			)
			return
		}
		if identity.Site.IsNull() || identity.Site.ValueString() == "" {
			identity.Site = types.StringValue(r.client.Site)
		}
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("id"), identity.ID)...)
		resp.Diagnostics.Append(
			resp.State.SetAttribute(ctx, path.Root("site"), identity.Site)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &identity)...)
		return
	}

	// Classic string import: "id", "site:id", "name=<name>", "site:name=<name>",
	// or a bare network name.
	importID := req.ID
	site := ""
	if idParts := strings.Split(importID, ":"); len(idParts) == 2 {
		site = idParts[0]
		importID = idParts[1]
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site"), site)...)
	}
	if site == "" {
		site = r.client.Site
	}

	if name, ok := strings.CutPrefix(importID, "name="); ok {
		// Import by name; Read resolves the ID and fills the identity.
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)
		return
	}

	if regexp.MustCompile(`^[0-9a-f]{24}$`).MatchString(importID) {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
		idModel := vpnServerIdentityModel{
			ID:   types.StringValue(importID),
			Site: types.StringValue(site),
		}
		resp.Diagnostics.Append(resp.Identity.Set(ctx, &idModel)...)
		return
	}

	// Fall back to import by name; Read resolves the ID and fills the identity.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), importID)...)
}

// modelToNetwork converts from Terraform model to unifi.Network.
func (r *vpnServerResource) modelToNetwork(
	ctx context.Context,
	model *vpnServerResourceModel,
) (*unifi.Network, diag.Diagnostics) {
	var diags diag.Diagnostics

	network := &unifi.Network{
		Name:              model.Name.ValueStringPointer(),
		Purpose:           unifi.PurposeUserVPN,
		Enabled:           boolPointerIfKnown(model.Enabled),
		IPSubnet:          model.Subnet.ValueStringPointer(),
		IPV6Subnet:        model.SubnetIPv6.ValueStringPointer(),
		SettingPreference: new("manual"),
	}

	// Determine VPN type from which nested block is configured
	hasWireguard := !model.Wireguard.IsNull() && !model.Wireguard.IsUnknown()
	hasL2TP := !model.L2TP.IsNull() && !model.L2TP.IsUnknown()
	hasOpenVPN := !model.OpenVPN.IsNull() && !model.OpenVPN.IsUnknown()

	switch {
	case hasWireguard:
		network.VPNType = new("wireguard-server")
	case hasL2TP:
		network.VPNType = new("l2tp-server")
	case hasOpenVPN:
		network.VPNType = new("openvpn-server")
	default:
		diags.AddError(
			"Missing VPN Type Configuration",
			"Exactly one of `wireguard`, `l2tp`, or `openvpn` must be specified.",
		)
		return nil, diags
	}

	// Handle DNS configuration (shared across all VPN types)
	if !model.DNS.IsNull() && !model.DNS.IsUnknown() {
		var dns vpnServerDNSModel
		d := model.DNS.As(ctx, &dns, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			if !dns.Enabled.IsNull() && !dns.Enabled.IsUnknown() {
				network.DHCPDDNSEnabled = dns.Enabled.ValueBool()
			}

			if !dns.Servers.IsNull() && !dns.Servers.IsUnknown() {
				var dnsServers []string
				d := dns.Servers.ElementsAs(ctx, &dnsServers, false)
				diags.Append(d...)
				if !diags.HasError() {
					if len(dnsServers) > 0 {
						network.DHCPDDNS1 = new(dnsServers[0])
						// Default enabled to true when servers are specified
						if dns.Enabled.IsNull() || dns.Enabled.IsUnknown() {
							network.DHCPDDNSEnabled = true
						}
					}
					if len(dnsServers) > 1 {
						network.DHCPDDNS2 = new(dnsServers[1])
					}
				}
			}
		}
	}

	// Handle WAN configuration (shared, but mapped to VPN-type-specific API fields)
	if !model.WAN.IsNull() && !model.WAN.IsUnknown() {
		var wan vpnServerWANModel
		d := model.WAN.As(ctx, &wan, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			switch {
			case hasWireguard:
				network.WireguardLocalWANIP = stringPointerIfKnown(wan.IP)
				network.WireguardInterface = stringPointerIfKnown(wan.Interface)
			case hasL2TP:
				network.L2TpLocalWANIP = stringPointerIfKnown(wan.IP)
				network.L2TpInterface = stringPointerIfKnown(wan.Interface)
			case hasOpenVPN:
				network.OpenVPNLocalWANIP = stringPointerIfKnown(wan.IP)
				network.OpenVPNInterface = stringPointerIfKnown(wan.Interface)
			}
		}
	}

	// RADIUS profile ID (applicable to L2TP and OpenVPN)
	if !model.RADIUSProfileID.IsNull() && !model.RADIUSProfileID.IsUnknown() {
		network.RADIUSProfileID = model.RADIUSProfileID.ValueStringPointer()
	}

	// Handle WireGuard-specific configuration
	if hasWireguard {
		var wireguard vpnServerWireguardModel
		d := model.Wireguard.As(ctx, &wireguard, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			// A null/unknown private_key leaves the field nil: Create and
			// Update resolve it (private_key_wo, the controller's current key,
			// or a generated one). See resolveWireguardPrivateKey.
			if !wireguard.PrivateKey.IsNull() && !wireguard.PrivateKey.IsUnknown() {
				network.WireguardPrivateKey = wireguard.PrivateKey.ValueStringPointer()
			}
			network.LocalPort = int64PointerIfKnown(wireguard.Port)
			network.WireguardInterfaceBindingModeIPVersion = stringPointerIfKnown(
				wireguard.IPVersion,
			)
		}
	}

	// Handle L2TP-specific configuration
	if hasL2TP {
		var l2tp vpnServerL2TPModel
		d := model.L2TP.As(ctx, &l2tp, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.L2TpAllowWeakCiphers = l2tp.AllowWeakCiphers.ValueBool()
			network.IPSecPreSharedKey = l2tp.PreSharedKey.ValueStringPointer()
		}
	}

	// Handle OpenVPN-specific configuration
	if hasOpenVPN {
		var openvpn vpnServerOpenVPNModel
		d := model.OpenVPN.As(ctx, &openvpn, basetypes.ObjectAsOptions{})
		diags.Append(d...)
		if !diags.HasError() {
			network.LocalPort = int64PointerIfKnown(openvpn.Port)
			network.OpenVPNMode = stringPointerIfKnown(openvpn.Mode)
			network.OpenVPNEncryptionCipher = stringPointerIfKnown(openvpn.EncryptionCipher)

			// Send computed certificate/key fields back on updates; on create
			// the not-yet-generated pairs go out as "" like the flat leaves did.
			var pd diag.Diagnostics
			network.ServerCrt, network.ServerKey, pd = vpnServerCertPairPointers(
				ctx,
				openvpn.Server,
			)
			diags.Append(pd...)
			network.DhKey = openvpn.DhKey.ValueStringPointer()
			network.SharedClientCrt, network.SharedClientKey, pd = vpnServerCertPairPointers(
				ctx,
				openvpn.SharedClient,
			)
			diags.Append(pd...)
			network.AuthKey = openvpn.AuthKey.ValueStringPointer()
			network.CaCrt, network.CaKey, pd = vpnServerCertPairPointers(ctx, openvpn.Ca)
			diags.Append(pd...)
		}
	}

	return network, diags
}

// vpnServerCreateDefaults fills the values a new server has always been
// created with when the configuration leaves them out. They used to be schema
// Defaults, but the framework applies a Default before plan modifiers run, so
// it also overwrote the controller's value on every adopted server (#544).
// Applying them here, on create only, keeps the create request unchanged while
// an existing server keeps what the controller holds.
//
// `enabled` is already planned `true` by planmodifiers.BoolDefaultOnCreate.
// The port stays a request-time fill-in: its default depends on the server
// type, and an attribute modifier on wireguard.port / openvpn.port would also
// plan it when an existing server switches type, where the controller keeps
// its current port.
func vpnServerCreateDefaults(network *unifi.Network) {
	networkCreateEnabled(network)
	if network.LocalPort == nil && network.VPNType != nil {
		switch *network.VPNType {
		case "wireguard-server":
			network.LocalPort = new(int64(51820))
		case "openvpn-server":
			network.LocalPort = new(int64(1194))
		}
	}
}

// networkToModel converts from unifi.Network to Terraform model.
func (r *vpnServerResource) networkToModel(
	ctx context.Context,
	network *unifi.Network,
	model *vpnServerResourceModel,
	site string,
	priorState *vpnServerResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(network.ID)
	model.Site = types.StringValue(site)
	model.Name = types.StringPointerValue(network.Name)
	model.Enabled = networkEnabled(network.Enabled)
	if network.IPSubnet != nil {
		model.Subnet = cidrtypes.NewIPv4PrefixValue(*network.IPSubnet)
	} else {
		model.Subnet = cidrtypes.NewIPv4PrefixNull()
	}
	if network.IPV6Subnet != nil && *network.IPV6Subnet != "" {
		model.SubnetIPv6 = cidrtypes.NewIPv6PrefixValue(*network.IPV6Subnet)
	} else {
		model.SubnetIPv6 = cidrtypes.NewIPv6PrefixNull()
	}

	// Build DNS nested object
	{
		var dnsServersList types.List
		dnsServers := collectNonEmptyStringPointers(network.DHCPDDNS1, network.DHCPDDNS2)
		if len(dnsServers) > 0 {
			var d diag.Diagnostics
			dnsServersList, d = types.ListValueFrom(ctx, types.StringType, dnsServers)
			diags.Append(d...)
		} else {
			dnsServersList = types.ListNull(types.StringType)
		}

		dnsValue := vpnServerDNSModel{
			Enabled: types.BoolValue(network.DHCPDDNSEnabled),
			Servers: dnsServersList,
		}
		var d diag.Diagnostics
		model.DNS, d = types.ObjectValueFrom(ctx, vpnServerDNSModel{}.AttributeTypes(), dnsValue)
		diags.Append(d...)
	}

	// Determine VPN type from the API response
	vpnType := ""
	if network.VPNType != nil {
		vpnType = *network.VPNType
	}

	// Build WAN nested object with VPN-type-specific API field mapping
	{
		var wanIP, wanIface *string
		switch vpnType {
		case "wireguard-server":
			wanIP = network.WireguardLocalWANIP
			wanIface = network.WireguardInterface
		case "l2tp-server":
			wanIP = network.L2TpLocalWANIP
			wanIface = network.L2TpInterface
		case "openvpn-server":
			wanIP = network.OpenVPNLocalWANIP
			wanIface = network.OpenVPNInterface
		}

		wanValue := vpnServerWANModel{
			IP:        types.StringPointerValue(wanIP),
			Interface: types.StringPointerValue(wanIface),
		}
		var d diag.Diagnostics
		model.WAN, d = types.ObjectValueFrom(ctx, vpnServerWANModel{}.AttributeTypes(), wanValue)
		diags.Append(d...)
	}

	// RADIUS profile ID
	if network.RADIUSProfileID != nil && *network.RADIUSProfileID != "" {
		model.RADIUSProfileID = types.StringPointerValue(network.RADIUSProfileID)
	} else {
		model.RADIUSProfileID = types.StringNull()
	}

	// Build VPN-type-specific nested objects
	switch vpnType {
	case "wireguard-server":
		apiPrivateKey := types.StringPointerValue(network.WireguardPrivateKey)
		privateKeyVal := apiPrivateKey
		privateKeyWOVersion := types.Int64Null()
		if priorState != nil && !priorState.Wireguard.IsNull() &&
			!priorState.Wireguard.IsUnknown() {
			var priorWG vpnServerWireguardModel
			d := priorState.Wireguard.As(ctx, &priorWG, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !diags.HasError() {
				switch {
				case priorWG.PrivateKey.IsNull():
					// A null private_key in prior state (or plan) means the
					// write-only alternative is in use. Never copy the
					// controller's echoed private key into state. (After an
					// import the whole wireguard object is null, so the key
					// is read back as before.)
					privateKeyVal = types.StringNull()
				case privateKeyVal.IsNull() || privateKeyVal.ValueString() == "":
					// Preserve private key from prior state if the API
					// doesn't return it.
					privateKeyVal = priorWG.PrivateKey
				}
				privateKeyWOVersion = priorWG.PrivateKeyWOVersion
			}
		}

		strPtrToType := func(ptr *string) types.String {
			if ptr == nil || *ptr == "" {
				return types.StringNull()
			}
			return types.StringValue(*ptr)
		}

		wireguardValue := vpnServerWireguardModel{
			PrivateKey:          privateKeyVal,
			PrivateKeyWO:        types.StringNull(),
			PrivateKeyWOVersion: privateKeyWOVersion,
			PublicKey:           strPtrToType(network.WireguardPublicKey),
			Port:                types.Int64PointerValue(network.LocalPort),
			// A server without the setting listens on IPv4.
			IPVersion: types.StringValue("v4"),
		}
		if v := network.WireguardInterfaceBindingModeIPVersion; v != nil && *v != "" {
			wireguardValue.IPVersion = types.StringValue(*v)
		}
		// The controller does not return the public key: derive it from the
		// key the controller holds (also when private_key is kept out of
		// state), falling back to the one in state.
		derivedFrom := apiPrivateKey
		if derivedFrom.IsNull() || derivedFrom.ValueString() == "" {
			derivedFrom = privateKeyVal
		}
		if wireguardValue.PublicKey.IsNull() && !derivedFrom.IsNull() &&
			!derivedFrom.IsUnknown() {
			if pub, err := wireGuardPublicKey(derivedFrom.ValueString()); err == nil {
				wireguardValue.PublicKey = types.StringValue(pub)
			}
		}
		var d diag.Diagnostics
		model.Wireguard, d = types.ObjectValueFrom(
			ctx,
			vpnServerWireguardModel{}.AttributeTypes(),
			wireguardValue,
		)
		diags.Append(d...)
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())

	case "l2tp-server":
		// Preserve pre-shared key from prior state since the API does not return it
		pskVal := types.StringPointerValue(network.IPSecPreSharedKey)
		if (pskVal.IsNull() || pskVal.ValueString() == "") &&
			priorState != nil && !priorState.L2TP.IsNull() && !priorState.L2TP.IsUnknown() {
			var priorL2TP vpnServerL2TPModel
			d := priorState.L2TP.As(ctx, &priorL2TP, basetypes.ObjectAsOptions{})
			diags.Append(d...)
			if !diags.HasError() {
				pskVal = priorL2TP.PreSharedKey
			}
		}

		l2tpValue := vpnServerL2TPModel{
			AllowWeakCiphers: types.BoolValue(network.L2TpAllowWeakCiphers),
			PreSharedKey:     pskVal,
		}
		var d diag.Diagnostics
		model.L2TP, d = types.ObjectValueFrom(ctx, vpnServerL2TPModel{}.AttributeTypes(), l2tpValue)
		diags.Append(d...)
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())

	case "openvpn-server":
		server, d := vpnServerCertPairObject(ctx, network.ServerCrt, network.ServerKey)
		diags.Append(d...)
		sharedClient, d := vpnServerCertPairObject(
			ctx,
			network.SharedClientCrt,
			network.SharedClientKey,
		)
		diags.Append(d...)
		ca, d := vpnServerCertPairObject(ctx, network.CaCrt, network.CaKey)
		diags.Append(d...)
		openvpnValue := vpnServerOpenVPNModel{
			Port:             types.Int64PointerValue(network.LocalPort),
			Mode:             types.StringPointerValue(network.OpenVPNMode),
			EncryptionCipher: types.StringPointerValue(network.OpenVPNEncryptionCipher),
			Server:           server,
			DhKey:            types.StringPointerValue(network.DhKey),
			SharedClient:     sharedClient,
			AuthKey:          types.StringPointerValue(network.AuthKey),
			Ca:               ca,
		}
		model.OpenVPN, d = types.ObjectValueFrom(
			ctx,
			vpnServerOpenVPNModel{}.AttributeTypes(),
			openvpnValue,
		)
		diags.Append(d...)
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())

	default:
		// Unknown VPN type — null out all type-specific blocks
		model.Wireguard = types.ObjectNull(vpnServerWireguardModel{}.AttributeTypes())
		model.L2TP = types.ObjectNull(vpnServerL2TPModel{}.AttributeTypes())
		model.OpenVPN = types.ObjectNull(vpnServerOpenVPNModel{}.AttributeTypes())
	}

	return diags
}

// ListResourceConfigSchema implements [list.ListResource].
func (r *vpnServerResource) ListResourceConfigSchema(
	ctx context.Context,
	req list.ListResourceSchemaRequest,
	resp *list.ListResourceSchemaResponse,
) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: "List VPN servers in a site.",
		Attributes: map[string]listschema.Attribute{
			"site": listschema.StringAttribute{
				MarkdownDescription: "The name of the site to list VPN servers from.",
				Optional:            true,
			},
		},
		Blocks: map[string]listschema.Block{
			"filter": listschema.ListNestedBlock{
				NestedObject: listschema.NestedBlockObject{
					Attributes: map[string]listschema.Attribute{
						"name": listschema.StringAttribute{
							MarkdownDescription: "The name of the filter to apply. Supported values are: `name`.",
							Required:            true,
						},
						"value": listschema.StringAttribute{
							MarkdownDescription: "The value to filter by.",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

// List implements [list.ListResource].
func (r *vpnServerResource) List(
	ctx context.Context,
	req list.ListRequest,
	stream *list.ListResultsStream,
) {
	var config vpnServerListConfigModel

	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	site := config.Site.ValueString()
	if site == "" {
		site = r.client.Site
	}

	// Process filter blocks.
	var filters []vpnServerListFilterModel
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		config.Filter.ElementsAs(ctx, &filters, false)
	}

	postFilters := make(map[string]string)
	for _, f := range filters {
		postFilters[f.Name.ValueString()] = f.Value.ValueString()
	}

	networks, err := r.client.ListNetwork(ctx, site)
	if err != nil {
		var d diag.Diagnostics
		d.AddError("Error Listing VPN Servers", "Could not list networks: "+err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(d)
		return
	}

	stream.Results = func(push func(list.ListResult) bool) {
		for _, network := range networks {
			// Filter by purpose: only remote-user-vpn networks.
			if network.Purpose != unifi.PurposeUserVPN {
				continue
			}

			// Apply name filter if specified.
			if nameFilter, ok := postFilters["name"]; ok {
				if network.Name == nil || *network.Name != nameFilter {
					continue
				}
			}

			result := req.NewListResult(ctx)
			if network.Name != nil {
				result.DisplayName = *network.Name
			}

			// Set identity.
			result.Diagnostics.Append(
				result.Identity.Set(ctx, vpnServerIdentityModel{
					ID:   types.StringValue(network.ID),
					Site: types.StringValue(site),
				})...,
			)

			// Convert to model.
			var model vpnServerResourceModel
			result.Diagnostics.Append(
				r.networkToModel(ctx, &network, &model, site, &vpnServerResourceModel{})...)
			if !result.Diagnostics.HasError() {
				model.Timeouts = timeoutsNullValue()
				result.Diagnostics.Append(result.Resource.Set(ctx, model)...)
			}

			if !push(result) {
				return
			}
		}
	}
}

// readPrivateKeyWO reads the write-only key from configuration. Write-only
// values are never available through plan or state.
func (r *vpnServerResource) readPrivateKeyWO(
	ctx context.Context,
	config tfsdk.Config,
	diags *diag.Diagnostics,
) types.String {
	var privateKeyWO types.String
	diags.Append(config.GetAttribute(
		ctx,
		path.Root("wireguard").AtName("private_key_wo"),
		&privateKeyWO,
	)...)
	return privateKeyWO
}

// vpnServerSendPrivateKeyWO reports whether an update must send
// private_key_wo: when private_key_wo_version changes, or when the server
// moves to the write-only key (the prior state held a private_key, or no
// WireGuard block at all). Otherwise the controller keeps its current key.
func vpnServerSendPrivateKeyWO(
	ctx context.Context,
	plan, prior *vpnServerResourceModel,
) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan.Wireguard.IsNull() || plan.Wireguard.IsUnknown() {
		return false, diags
	}
	if prior == nil || prior.Wireguard.IsNull() || prior.Wireguard.IsUnknown() {
		return true, diags
	}
	var planWG, priorWG vpnServerWireguardModel
	diags.Append(plan.Wireguard.As(ctx, &planWG, basetypes.ObjectAsOptions{})...)
	diags.Append(prior.Wireguard.As(ctx, &priorWG, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return false, diags
	}
	if !priorWG.PrivateKey.IsNull() {
		return true, diags
	}
	return !planWG.PrivateKeyWOVersion.Equal(priorWG.PrivateKeyWOVersion), diags
}

// resolveWireguardPrivateKey fills network.WireguardPrivateKey for a
// WireGuard server when private_key did not set it: privateKeyWO when given;
// otherwise, on update, the key the controller currently holds (so an update
// never drops or rotates it); otherwise a freshly generated key (the
// controller does not generate one and rejects a server without a key with
// api.err.WireguardMissingPrivateKey).
func (r *vpnServerResource) resolveWireguardPrivateKey(
	ctx context.Context,
	network *unifi.Network,
	site, id string,
	privateKeyWO types.String,
	create bool,
) diag.Diagnostics {
	var diags diag.Diagnostics
	if network.VPNType == nil || *network.VPNType != "wireguard-server" {
		return diags
	}
	if !privateKeyWO.IsNull() && !privateKeyWO.IsUnknown() {
		network.WireguardPrivateKey = privateKeyWO.ValueStringPointer()
		return diags
	}
	if network.WireguardPrivateKey != nil {
		return diags
	}
	if !create && id != "" {
		current, err := r.client.GetNetwork(ctx, site, id)
		if err != nil {
			diags.AddError(
				"Error Reading VPN Server",
				"Could not read the current WireGuard private key of VPN server ID "+
					id+": "+err.Error(),
			)
			return diags
		}
		if current.WireguardPrivateKey != nil && *current.WireguardPrivateKey != "" {
			network.WireguardPrivateKey = current.WireguardPrivateKey
			return diags
		}
	}
	key, err := generateWireGuardPrivateKey()
	if err != nil {
		diags.AddError("Unable to generate WireGuard private key", err.Error())
		return diags
	}
	network.WireguardPrivateKey = &key
	return diags
}

// vpnServerPrivateKeyNullWithWO plans wireguard.private_key as null when
// private_key_wo is configured, so neither key reaches the state. It also
// plans the removal of a key kept in state from before the switch (for
// example after an import), and keeps a key that is out of state out of it.
type vpnServerPrivateKeyNullWithWO struct{}

func (vpnServerPrivateKeyNullWithWO) Description(context.Context) string {
	return "Plans private_key as null when private_key_wo is configured."
}

func (m vpnServerPrivateKeyNullWithWO) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (vpnServerPrivateKeyNullWithWO) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	var wo types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(
		ctx,
		req.Path.ParentPath().AtName("private_key_wo"),
		&wo,
	)...)
	if !wo.IsNull() {
		resp.PlanValue = types.StringNull()
		return
	}
	// Once kept out of state, the key stays out after private_key_wo is
	// removed from the configuration (the controller keeps its key), until a
	// private_key is configured explicitly.
	if req.ConfigValue.IsNull() && req.StateValue.IsNull() && !req.State.Raw.IsNull() {
		var stateWG types.Object
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, req.Path.ParentPath(), &stateWG)...)
		if !stateWG.IsNull() && !stateWG.IsUnknown() {
			resp.PlanValue = types.StringNull()
		}
	}
}

// vpnServerPublicKeyUnknownOnKeyChange plans wireguard.public_key as unknown
// when the private key is about to change: a new private_key, or a
// private_key_wo sent on this update (see vpnServerSendPrivateKeyWO).
type vpnServerPublicKeyUnknownOnKeyChange struct{}

func (vpnServerPublicKeyUnknownOnKeyChange) Description(context.Context) string {
	return "Plans public_key as unknown when the private key changes."
}

func (m vpnServerPublicKeyUnknownOnKeyChange) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (vpnServerPublicKeyUnknownOnKeyChange) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() || resp.PlanValue.IsUnknown() {
		return
	}
	parent := req.Path.ParentPath()
	var planKey, stateKey types.String
	var planVersion, stateVersion types.Int64
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, parent.AtName("private_key"), &planKey)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, parent.AtName("private_key"), &stateKey)...)
	resp.Diagnostics.Append(
		req.Plan.GetAttribute(ctx, parent.AtName("private_key_wo_version"), &planVersion)...)
	resp.Diagnostics.Append(
		req.State.GetAttribute(ctx, parent.AtName("private_key_wo_version"), &stateVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}
	keyChanged := !planKey.IsUnknown() && !planKey.Equal(stateKey)
	if keyChanged || !planVersion.Equal(stateVersion) {
		resp.PlanValue = types.StringUnknown()
	}
}

// generateWireGuardPrivateKey returns a fresh base64-encoded Curve25519 private
// key in the WireGuard format. The UniFi controller does not generate a key for
// a WireGuard VPN server (it rejects creation with api.err.WireguardMissingPrivateKey),
// so the provider generates one when the user does not supply it.
func generateWireGuardPrivateKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	// Curve25519 clamping, per the WireGuard key format.
	key[0] &= 248
	key[31] &= 127
	key[31] |= 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// wireGuardPublicKey returns the base64-encoded public key of a WireGuard
// private key.
func wireGuardPublicKey(privateKey string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", err
	}
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}
