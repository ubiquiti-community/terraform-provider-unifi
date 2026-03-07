package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
	"github.com/iancoleman/strcase"
	"github.com/ubiquiti-community/go-unifi/internal/fields"
)

type replacement struct {
	Old string
	New string
}

var fieldReps = []replacement{
	{"Dhcpdv6", "DHCPDV6"},

	{"Dhcpd", "DHCPD"},
	{"Idx", "IDX"},
	{"Ipsec", "IPSec"},
	{"Ipv6", "IPV6"},
	{"Openvpn", "OpenVPN"},
	{"Tftp", "TFTP"},
	{"Wlangroup", "WLANGroup"},

	{"FrrBgpdConfig", "Config"},
	{"BgpConfig", "BGPConfig"},

	{"Bc", "Broadcast"},
	{"Dhcp", "DHCP"},
	{"Dns", "DNS"},
	{"Dpi", "DPI"},
	{"Dtim", "DTIM"},
	{"Firewallgroup", "FirewallGroup"},
	{"Fixedip", "FixedIP"},
	{"Icmp", "ICMP"},
	{"Id", "ID"},
	{"Igmp", "IGMP"},
	{"Ip", "IP"},
	{"Leasetime", "LeaseTime"},
	{"Mac", "MAC"},
	{"Mcastenhance", "MulticastEnhance"},
	{"Minrssi", "MinRSSI"},
	{"Monthdays", "MonthDays"},
	{"Nat", "NAT"},
	{"Networkconf", "Network"},
	{"Networkgroup", "NetworkGroup"},
	{"Pd", "PD"},
	{"Pmf", "PMF"},
	{"pnp", "PnP"},
	{"Portconf", "PortProfile"},
	{"Qos", "QOS"},
	{"Radiusprofile", "RADIUSProfile"},
	{"Radius", "RADIUS"},
	{"Ssid", "SSID"},
	{"Smartq", "SmartQ"},
	{"Startdate", "StartDate"},
	{"Starttime", "StartTime"},
	{"Stopdate", "StopDate"},
	{"Stoptime", "StopTime"},
	{"Supression", "Suppression"}, //nolint:misspell
	{"Tcp", "TCP"},
	{"Udp", "UDP"},
	{"Usergroup", "UserGroup"},
	{"Utc", "UTC"},
	{"Vlan", "VLAN"},
	{"Vpn", "VPN"},
	{"Wan", "WAN"},
	{"Wep", "WEP"},
	{"Wlan", "WLAN"},
	{"Wpa", "WPA"},
	{"XWireguardPrivateKey", "WireguardPrivateKey"},
	{"XSsh", "SSH"},
	{"XMgmt", "Mgmt"},
	{"UnifiIDp", "UniFiIdentityProvider"},
}

var fileReps = []replacement{
	{"WlanConf", "WLAN"},
	{"Dhcp", "DHCP"},
	{"Wlan", "WLAN"},
	{"NetworkConf", "Network"},
	{"PortConf", "PortProfile"},
	{"RadiusProfile", "RADIUSProfile"},
	{"ApGroups", "APGroup"},
	{"DnsRecord", "DNSRecord"},
	{"BgpConfig", "BGPConfig"},
	{"User", "Client"},
	{"UserGroup", "ClientGroup"},
}

type ResourceInfo struct {
	StructName     string
	ResourcePath   string
	Types          map[string]*FieldInfo
	FieldProcessor func(name string, f *FieldInfo) error
}

type FieldInfo struct {
	FieldName           string
	JSONName            string
	FieldType           string
	IsPointer           bool
	FieldValidation     string
	OmitEmpty           bool
	IsArray             bool
	Fields              map[string]*FieldInfo
	CustomUnmarshalType string
	CustomUnmarshalFunc string
}

func NewResource(structName string, resourcePath string) *ResourceInfo {
	baseType := NewFieldInfo(structName, resourcePath, "struct", "", false, false, false, "")
	resource := &ResourceInfo{
		StructName:   structName,
		ResourcePath: resourcePath,
		Types: map[string]*FieldInfo{
			structName: baseType,
		},
		FieldProcessor: func(name string, f *FieldInfo) error { return nil },
	}

	// Since template files iterate through map keys in sorted order, these initial fields
	// are named such that they stay at the top for consistency. The spacer items create a
	// blank line in the resulting generated file.
	//
	// This hack is here for stability of the generatd code, but can be removed if desired.
	baseType.Fields = map[string]*FieldInfo{
		"   ID":      NewFieldInfo("ID", "_id", fields.String, "", true, false, false, ""),
		"   SiteID":  NewFieldInfo("SiteID", "site_id", fields.String, "", true, false, false, ""),
		"   _Spacer": nil,

		"  Hidden":   NewFieldInfo("Hidden", "attr_hidden", fields.Bool, "", true, false, false, ""),
		"  HiddenID": NewFieldInfo("HiddenID", "attr_hidden_id", fields.String, "", true, false, false, ""),
		"  NoDelete": NewFieldInfo("NoDelete", "attr_no_delete", fields.Bool, "", true, false, false, ""),
		"  NoEdit":   NewFieldInfo("NoEdit", "attr_no_edit", fields.Bool, "", true, false, false, ""),
		"  _Spacer":  nil,

		" _Spacer": nil,
	}

	switch {
	case resource.IsSetting():
		resource.ResourcePath = strcase.ToSnake(strings.TrimPrefix(structName, "Setting"))
		baseType.Fields[" Key"] = NewFieldInfo("Key", "key", fields.String, "", false, false, false, "")
		if resource.StructName == "SettingUsg" {
			// Removed in v7, retaining for backwards compatibility
			baseType.Fields["MdnsEnabled"] = NewFieldInfo("MdnsEnabled", "mdns_enabled", fields.Bool, "", false, false, false, "")
		}
	case resource.StructName == "DNSRecord":
		resource.ResourcePath = "static-dns"
	case resource.StructName == "FirewallZone":
		resource.ResourcePath = "firewall/zone"
	case resource.StructName == "OSPFRouter":
		resource.ResourcePath = "ospf/router"
	case resource.StructName == "FirewallPolicy":
		resource.ResourcePath = "firewall-policies"
	case resource.StructName == "TrafficRoute":
		resource.ResourcePath = "trafficroutes"
	case resource.StructName == "Network":
		baseType.Fields["WANEgressQOSEnabled"] = NewFieldInfo("WANEgressQOSEnabled", "wan_egress_qos_enabled", fields.Bool, "", true, false, true, "")
		baseType.Fields["UPnPEnabled"] = NewFieldInfo("UPnPEnabled", "upnp_enabled", fields.Bool, "", true, false, true, "")
		baseType.Fields["UPnPWANInterface"] = NewFieldInfo("UPnPWANInterface", "upnp_wan_interface", fields.String, "", true, false, true, "")
		baseType.Fields["UPnPNatPMPEnabled"] = NewFieldInfo("UPnPNatPMPEnabled", "upnp_nat_pmp_enabled", fields.Bool, "", true, false, true, "")
		baseType.Fields["UPnPSecureMode"] = NewFieldInfo("UPnPSecureMode", "upnp_secure_mode", fields.Bool, "", true, false, true, "")
		baseType.Fields["IPAliases"] = NewFieldInfo("IPAliases", "ip_aliases", fields.String, "", true, true, false, "")
		baseType.Fields["DHCPRelayServers"] = NewFieldInfo("DHCPRelayServers", "dhcp_relay_servers", fields.String, "", true, true, false, "")
	case resource.StructName == "Device":
		baseType.Fields["PortTable"] = NewFieldInfo("PortTable", "port_table", "[]DevicePortTable", "", true, false, false, "")
		baseType.Fields[" MAC"] = NewFieldInfo("MAC", "mac", fields.String, "", true, false, false, "")
		baseType.Fields["Adopted"] = NewFieldInfo("Adopted", "adopted", fields.Bool, "", false, false, false, "")
		baseType.Fields["Model"] = NewFieldInfo("Model", "model", fields.String, "", true, false, false, "")
		baseType.Fields["State"] = NewFieldInfo("State", "state", "DeviceState", "", false, false, false, "")
		baseType.Fields["Type"] = NewFieldInfo("Type", "type", fields.String, "", true, false, false, "")
		baseType.Fields["InformIP"] = NewFieldInfo("InformIP", "inform_ip", fields.String, "", true, false, false, "")
		baseType.Fields["IP"] = NewFieldInfo("IP", "ip", fields.String, "", true, false, false, "")
	case resource.StructName == "Client":
		baseType.Fields[" DisplayName"] = NewFieldInfo("DisplayName", "display_name", fields.String, "non-generated field", true, false, false, "")
	case resource.StructName == "WLAN":
		// this field removed in v6, retaining for backwards compatibility
		baseType.Fields["WLANGroupID"] = NewFieldInfo("WLANGroupID", "wlangroup_id", fields.String, "", false, false, false, "")
	case resource.StructName == "BGPConfig":
		resource.ResourcePath = "bgp/config"
	}

	return resource
}

func NewFieldInfo(
	fieldName string,
	jsonName string,
	fieldType string,
	fieldValidation string,
	omitempty bool,
	isArray bool,
	isPointer bool,
	customUnmarshalType string,
) *FieldInfo {
	return &FieldInfo{
		FieldName:           fieldName,
		JSONName:            jsonName,
		FieldType:           fieldType,
		FieldValidation:     fieldValidation,
		OmitEmpty:           omitempty,
		IsArray:             isArray,
		IsPointer:           isPointer,
		CustomUnmarshalType: customUnmarshalType,
	}
}

func cleanName(name string, reps []replacement) string {
	for _, rep := range reps {
		name = strings.ReplaceAll(name, rep.Old, rep.New)
	}

	return name
}

func usage() {
	fmt.Printf("Usage: %s [OPTIONS] version\n", path.Base(os.Args[0]))
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	outputDirFlag := flag.String(
		"output-dir",
		"unifi",
		"The output directory of the generated Go code",
	)
	downloadOnly := flag.Bool(
		"download-only",
		false,
		"Only download and build the fields JSON directory, do not generate",
	)
	useLatestVersion := flag.Bool("latest", false, "Use the latest available version")
	generateSpec := flag.Bool(
		"generate-spec",
		false,
		"Generate Terraform provider specification JSON file",
	)
	specOutputPath := flag.String(
		"spec-output",
		"specification.json",
		"Output path for the Terraform provider specification JSON file",
	)

	flag.Parse()

	specifiedVersion := flag.Arg(0)
	if specifiedVersion != "" && *useLatestVersion {
		fmt.Print("error: cannot specify version with latest\n\n")
		usage()
		os.Exit(1)
	} else if specifiedVersion == "" && !*useLatestVersion {
		fmt.Print("error: must specify version or latest\n\n")
		usage()
		os.Exit(1)
	}

	var unifiVersion *version.Version
	var unifiDownloadUrl *url.URL
	var err error

	if *useLatestVersion {
		unifiVersion, unifiDownloadUrl, err = latestUnifiVersion()
		if err != nil {
			panic(err)
		}
	} else {
		unifiVersion, err = version.NewVersion(specifiedVersion)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		unifiDownloadUrl, err = url.Parse(fmt.Sprintf("https://dl.ui.com/unifi/%s/unifi_sysvinit_all.deb", unifiVersion))
		if err != nil {
			panic(err)
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get the current filename")
	}

	versionBaseDir := filepath.Dir(filename)

	fieldsDir := filepath.Join(versionBaseDir, fmt.Sprintf("v%s", unifiVersion))

	outDir := filepath.Join(wd, *outputDirFlag)

	fieldsInfo, err := os.Stat(fieldsDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}

		err = os.MkdirAll(fieldsDir, 0o755)
		if err != nil {
			panic(err)
		}

		// download fields, create
		jarFile, err := downloadJar(unifiDownloadUrl, fieldsDir)
		if err != nil {
			panic(err)
		}

		err = extractJSON(jarFile, fieldsDir)
		if err != nil {
			panic(err)
		}

		// defer func() {
		// 	err = os.RemoveAll(fieldsDir)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// }()

		err = copyCustom(fieldsDir)
		if err != nil {
			panic(err)
		}

		fieldsInfo, err = os.Stat(fieldsDir)
		if err != nil {
			panic(err)
		}
	}
	if !fieldsInfo.IsDir() {
		panic("version info isn't a directory")
	}

	if *downloadOnly {
		fmt.Println("Fields JSON ready!")
		os.Exit(0)
	}

	fieldsFiles, err := os.ReadDir(fieldsDir)
	if err != nil {
		panic(err)
	}

	// Initialize specification generator
	specGen := NewSpecificationGenerator("unifi")

	for _, fieldsFile := range fieldsFiles {
		name := fieldsFile.Name()
		ext := filepath.Ext(name)

		switch name {
		case "AuthenticationRequest.json", "Setting.json", "Wall.json":
			continue
		}

		if filepath.Ext(name) != ".json" {
			continue
		}

		name = name[:len(name)-len(ext)]

		urlPath := strings.ToLower(name)
		structName := cleanName(name, fileReps)

		// For settings, create a cleaner filename without "setting_" prefix
		goFile := strcase.ToSnake(structName) + ".generated.go"
		if after, ok0 := strings.CutPrefix(structName, "Setting"); ok0 {
			// Remove "Setting" prefix for the file name
			cleanStructName := after
			goFile = strcase.ToSnake(cleanStructName) + ".generated.go"
		}
		fieldsFilePath := filepath.Join(fieldsDir, fieldsFile.Name())
		b, err := os.ReadFile(fieldsFilePath)
		if err != nil {
			fmt.Printf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}

		resource := NewResource(structName, urlPath)

		switch resource.StructName {
		case "Account":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "IP", "NetworkID":
					f.OmitEmpty = true
				}
				return nil
			}
		case "ChannelPlan":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "Channel", "BackupChannel", "TxPower":
					if f.FieldType == fields.String {
						f.CustomUnmarshalType = fields.Number
					}
				}
				return nil
			}
		case "Device":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "X", "Y":
					f.FieldType = "float64"
				case "StpPriority":
					f.FieldType = fields.Int
					f.CustomUnmarshalType = fields.Number
				case "ConfigNetwork", "EtherLighting", "MbbOverrides", "NutServer", "RpsOverride", "QOSProfile":
					f.IsPointer = true
				case "Ht":
					// Field within DeviceRadioTable nested type
					f.CustomUnmarshalType = "types.Number"
					f.CustomUnmarshalFunc = "types.ToInt64Pointer"
				}

				f.OmitEmpty = true
				switch name {
				case "PortOverrides":
					f.OmitEmpty = false
				}

				return nil
			}
		case "Network":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "InternetAccessEnabled", "IntraNetworkAccessEnabled":
					if f.FieldType == fields.Bool {
						f.CustomUnmarshalType = "*bool"
						f.CustomUnmarshalFunc = "emptyBoolToTrue"
					}
				case "IPSecEspLifetime", "IPSecIkeLifetime":
					f.FieldType = fields.Int
					f.IsPointer = true
				case "WANDNS1", "WANDNS2", "WANIPV6DNS1", "WANIPV6DNS2", "DHCPDStart", "DHCPDStop", "DHCPDUnifiController",
					"DHCPDTFTPServer", "DHCPDWins1", "DHCPDWins2", "DHCPDWPAdUrl", "DomainName", "DHCPDGateway", "DHCPDNtp1", "DHCPDNtp2":
					f.OmitEmpty = true
					f.IsPointer = true
				case "Purpose":
					f.OmitEmpty = false
					f.IsPointer = false
				}
				if f.OmitEmpty && !f.IsArray {
					switch f.FieldType {
					case fields.Bool, fields.String:
						f.IsPointer = true
					}
				}
				return nil
			}
		case "SettingGlobalAp":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				if strings.HasPrefix(name, "6E") {
					f.FieldName = strings.Replace(f.FieldName, "6E", "SixE", 1)
				}

				return nil
			}
		case "SettingMgmt":
			sshKeyField := NewFieldInfo(resource.StructName+"SSHKeys", "x_ssh_keys", "struct", "", false, false, false, "")
			sshKeyField.Fields = map[string]*FieldInfo{
				"name":        NewFieldInfo("Name", "name", fields.String, "", false, false, false, ""),
				"keyType":     NewFieldInfo("KeyType", "type", fields.String, "", false, false, false, ""),
				"key":         NewFieldInfo("Key", "key", fields.String, "", false, false, false, ""),
				"comment":     NewFieldInfo("Comment", "comment", fields.String, "", false, false, false, ""),
				"date":        NewFieldInfo("Date", "date", fields.String, "", false, false, false, ""),
				"fingerprint": NewFieldInfo("Fingerprint", "fingerprint", fields.String, "", false, false, false, ""),
			}
			resource.Types[sshKeyField.FieldName] = sshKeyField

			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				if name == "SSHKeys" {
					f.FieldType = sshKeyField.FieldName
				}
				return nil
			}
		case "SettingUsg":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				if strings.HasSuffix(name, "Timeout") && name != "ArpCacheTimeout" {
					f.FieldType = fields.Int
					f.CustomUnmarshalType = fields.Number
				}
				return nil
			}
		case "Nat":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "SourceFilter":
					f.IsPointer = true
				case "DestinationFilter":
					f.IsPointer = true
				}
				return nil
			}
		case "Client":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "Blocked":
					f.FieldType = fields.Bool
					f.IsPointer = true
				case "VirtualNetworkOverrideEnabled":
					f.FieldType = fields.Bool
					f.IsPointer = true
					f.OmitEmpty = true
				case "LastSeen":
					f.FieldType = fields.Int
					f.IsPointer = true
				}
				return nil
			}
		case "WLAN":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "ScheduleWithDuration":
					// always send schedule, so we can empty it if we want to
					f.OmitEmpty = false
				}
				return nil
			}
		case "DNSRecord":
			resource.FieldProcessor = func(name string, f *FieldInfo) error {
				switch name {
				case "Hidden", "NoDelete", "NoEdit", "Enabled":
					f.FieldType = fields.Bool
				case "Priority", "Ttl", "Weight":
					f.FieldType = fields.Int
					f.CustomUnmarshalType = fields.Number
				}
				return nil
			}
		}

		err = resource.processJSON(b)
		if err != nil {
			fmt.Printf("skipping file %s: %s", fieldsFile.Name(), err)
			continue
		}

		// Add resource to specification generator
		specGen.AddResource(resource)

		var code string
		if code, err = resource.generateCode(false); err != nil {
			panic(err)
		}

		// Determine output directory based on whether it's a setting
		var targetDir string
		if resource.IsSetting() {
			targetDir = filepath.Join(outDir, "settings")
			// Ensure settings directory exists
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				panic(err)
			}
		} else {
			targetDir = outDir
		}

		_ = os.Remove(filepath.Join(targetDir, goFile))
		if err := os.WriteFile(filepath.Join(targetDir, goFile), ([]byte)(code), 0o644); err != nil {
			panic(err)
		}

		if !resource.IsSetting() {
			implFile := strcase.ToSnake(structName) + ".go"
			implFilePath := filepath.Join(targetDir, implFile)

			if _, err := os.Stat(implFilePath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					var implCode string
					if implCode, err = resource.generateCode(true); err != nil {
						panic(err)
					}

					if err := os.WriteFile(filepath.Join(implFilePath), ([]byte)(implCode), 0o644); err != nil {
						panic(err)
					}
				}
			}
		}
	}

	// Write version file.
	versionGo := fmt.Appendf(nil, `
// Generated code. DO NOT EDIT.

package unifi

const UnifiVersion = %q
`, unifiVersion)

	versionGo, err = format.Source(versionGo)
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(filepath.Join(outDir, "version.generated.go"), versionGo, 0o644); err != nil {
		panic(err)
	}

	// Generate Terraform provider specification if requested
	if *generateSpec {
		specOutputFile := *specOutputPath
		if !filepath.IsAbs(specOutputFile) {
			specOutputFile = filepath.Join(wd, specOutputFile)
		}
		if err := specGen.WriteSpecification(specOutputFile); err != nil {
			panic(err)
		}
		fmt.Printf("Generated specification: %s\n", specOutputFile)
	}

	fmt.Printf("%s\n", outDir)
}

func (r *ResourceInfo) IsSetting() bool {
	return strings.HasPrefix(r.StructName, "Setting")
}

func (r *ResourceInfo) IsDevice() bool {
	return r.StructName == "Device"
}

func (r *ResourceInfo) IsV2() bool {
	return slices.Contains([]string{
		"ApGroup",
		"BGPConfig",
		"DNSRecord",
		"FirewallPolicy",
		"FirewallZone",
		"Nat",
		"OSPFRouter",
		"TrafficRoute",
	}, r.StructName)
}

func (r *ResourceInfo) CleanStructName() string {
	if r.IsSetting() {
		return strings.TrimPrefix(r.StructName, "Setting")
	}
	return r.StructName
}

func (r *ResourceInfo) processFields(fields map[string]any) {
	t := r.Types[r.StructName]
	for name, validation := range fields {
		fieldInfo, err := r.fieldInfoFromValidation(name, validation)
		if err != nil {
			continue
		}

		t.Fields[fieldInfo.FieldName] = fieldInfo
	}
}

func (r *ResourceInfo) fieldInfoFromValidation(name string, validation any) (*FieldInfo, error) {
	fieldName := strcase.ToCamel(name)
	fieldName = cleanName(fieldName, fieldReps)

	empty := &FieldInfo{}
	var fieldInfo *FieldInfo

	switch validation := validation.(type) {
	case []any:
		if len(validation) == 0 {
			fieldInfo = NewFieldInfo(fieldName, name, fields.String, "", false, true, false, "")
			err := r.FieldProcessor(fieldName, fieldInfo)
			return fieldInfo, err
		}
		if len(validation) > 1 {
			return empty, fmt.Errorf("unknown validation %#v", validation)
		}

		fieldInfo, err := r.fieldInfoFromValidation(name, validation[0])
		if err != nil {
			return empty, err
		}

		fieldInfo.OmitEmpty = true
		fieldInfo.IsArray = true
		fieldInfo.IsPointer = false

		err = r.FieldProcessor(fieldName, fieldInfo)
		return fieldInfo, err

	case map[string]any:
		typeName := r.StructName + fieldName

		result := NewFieldInfo(fieldName, name, typeName, "", true, false, true, "")
		result.Fields = make(map[string]*FieldInfo)

		for name, fv := range validation {
			child, err := r.fieldInfoFromValidation(name, fv)
			if err != nil {
				return empty, err
			}

			result.Fields[child.FieldName] = child
		}

		err := r.FieldProcessor(fieldName, result)
		r.Types[typeName] = result
		return result, err

	case string:
		fieldValidation := validation
		normalized := normalizeValidation(validation)

		omitEmpty := false

		switch normalized {
		case "falsetrue", "truefalse":
			fieldInfo = NewFieldInfo(fieldName, name, fields.Bool, "", omitEmpty, false, false, "")
			return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
		default:
			if _, err := strconv.ParseFloat(normalized, 64); err == nil {
				if normalized == "09" || normalized == "09.09" {
					fieldValidation = ""
				}

				if strings.Contains(normalized, ".") {
					if strings.Contains(validation, "\\.){3}") {
						break
					}

					omitEmpty = true
					fieldInfo = NewFieldInfo(fieldName, name, "float64", fieldValidation, omitEmpty, false, false, "")
					return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
				}

				omitEmpty = true
				fieldInfo = NewFieldInfo(fieldName, name, fields.Int, fieldValidation, omitEmpty, false, true, "")
				// fieldInfo.CustomUnmarshalType = fields.Number
				return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
			}
		}
		if validation != "" && normalized != "" {
			fmt.Printf("normalize %q to %q\n", validation, normalized)
		}

		omitEmpty = omitEmpty || (!strings.Contains(validation, "^$") && !strings.HasSuffix(fieldName, "Id"))
		fieldInfo = NewFieldInfo(fieldName, name, fields.String, fieldValidation, omitEmpty, false, false, "")
		return fieldInfo, r.FieldProcessor(fieldName, fieldInfo)
	}

	return empty, fmt.Errorf("unable to determine type from validation %q", validation)
}

func (r *ResourceInfo) processJSON(b []byte) error {
	var fields map[string]any
	err := json.Unmarshal(b, &fields)
	if err != nil {
		return err
	}

	r.processFields(fields)

	return nil
}

//go:embed api.go.tmpl
var apiGoTemplate string

//go:embed client.go.tmpl
var clientGoTemplate string

func (r *ResourceInfo) generateCode(isImpl bool) (string, error) {
	var err error
	var buf bytes.Buffer
	writer := io.Writer(&buf)

	var tpl *template.Template
	funcMap := template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}

	if isImpl {
		tpl = template.Must(template.New("client.go.tmpl").Funcs(funcMap).Parse(clientGoTemplate))
	} else {
		tpl = template.Must(template.New("api.go.tmpl").Funcs(funcMap).Parse(apiGoTemplate))
	}

	err = tpl.Execute(writer, r)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format source: %w", err)
	}

	return string(src), err
}

func normalizeValidation(re string) string {
	re = strings.ReplaceAll(re, "\\d", "[0-9]")
	re = strings.ReplaceAll(re, "[-+]?", "")
	re = strings.ReplaceAll(re, "[+-]?", "")
	re = strings.ReplaceAll(re, "[-]?", "")
	re = strings.ReplaceAll(re, "\\.", ".")
	re = strings.ReplaceAll(re, "[.]?", ".")

	quants := regexp.MustCompile(`\{\d*,?\d*\}|\*|\+|\?`)
	re = quants.ReplaceAllString(re, "")

	control := regexp.MustCompile(`[\(\[\]\)\|\-\$\^]`)
	re = control.ReplaceAllString(re, "")

	re = strings.TrimPrefix(re, "^")
	re = strings.TrimSuffix(re, "$")

	return re
}
