package unifi

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// fakeSettingDocuments are the site setting documents of the fake controller,
// keyed by setting key. There are deliberately no country or connectivity
// documents: a section whose document is absent must import as null. dpi and
// rsyslogd hold values other than the provider's leaf defaults.
var fakeSettingDocuments = map[string]string{
	"dpi": `{"key":"dpi","enabled":true,"fingerprintingEnabled":true}`,
	"rsyslogd": `{"key":"rsyslogd","enabled":true,"debug":true,"ip":"192.0.2.14","port":514,
		"log_all_contents":true,"netconsole_enabled":false,"netconsole_host":"192.0.2.15",
		"netconsole_port":6666,"this_controller":true,
		"this_controller_encrypted_only":true}`,
	"ntp": `{"key":"ntp","ntp_server_1":"time.cloudflare.com","ntp_server_2":"",
		"ntp_server_3":"","ntp_server_4":"","setting_preference":"manual"}`,
	"mgmt": `{"key":"mgmt","advanced_feature_enabled":true,"auto_upgrade":false,
		"auto_upgrade_hour":3,"debug_tools_enabled":true,"direct_connect_enabled":false,
		"unifi_idp_enabled":false,"wifiman_enabled":true,"x_ssh_enabled":true,
		"x_ssh_username":"admin","x_ssh_auth_password_enabled":true,"x_ssh_keys":[],
		"x_ssh_password":"stored-hash"}`,
	"igmp_snooping": `{"key":"igmp_snooping","enabled":true,"network_ids":["lan-a"]}`,
	"global_switch": `{"key":"global_switch","acl_device_isolation":[],"stp_version":"rstp",
		"dhcp_snoop":true,"jumboframe_enabled":false,"dot1x_portctrl_enabled":false}`,
	"usg": `{"key":"usg","broadcast_ping":false,"ftp_module":true,"gre_module":true,
		"h323_module":true,"pptp_module":true,"sip_module":true,"tftp_module":true,
		"receive_redirects":false,"send_redirects":true,"syn_cookies":true,
		"mss_clamp":"auto","offload_accounting":true,"offload_l2_blocking":true,
		"offload_sch":true,"timeout_setting_preference":"auto","upnp_enabled":false,
		"dns_verification":{"domain":"","primary_dns_server":"","secondary_dns_server":"",
		"setting_preference":"auto"}}`,
}

// fakeSettingController is a minimal fake controller (old-style API: 302 on
// /, cookie login at /api/login) serving the site setting documents in docs.
// A key in unreadable is listed but its own document cannot be read, and
// failList makes listing the settings fail. Every PUT's setting key is
// recorded in puts.
type fakeSettingController struct {
	docs       map[string]string
	unreadable map[string]bool
	failList   bool
	puts       []string
}

func newSettingFakeControllerClient(t *testing.T, fc *fakeSettingController) *Client {
	t.Helper()

	reply := func(w http.ResponseWriter, docs ...string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"rc":"ok"},"data":[` + strings.Join(docs, ",") + `]}`))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/manage", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "unifises", Value: "fake-session", Path: "/"})
		reply(w)
	})
	mux.HandleFunc("/api/s/default/get/setting", func(w http.ResponseWriter, r *http.Request) {
		if fc.failList {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"meta":{"rc":"error","msg":"api.err.Invalid"},"data":[]}`))
			return
		}
		keys := make([]string, 0, len(fc.docs))
		for key := range fc.docs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		docs := make([]string, 0, len(keys))
		for _, key := range keys {
			docs = append(docs, fc.docs[key])
		}
		reply(w, docs...)
	})
	mux.HandleFunc("/api/s/default/get/setting/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/s/default/get/setting/")
		if doc, ok := fc.docs[key]; ok && !fc.unreadable[key] {
			reply(w, doc)
			return
		}
		reply(w)
	})
	mux.HandleFunc("/api/s/default/set/setting/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/s/default/set/setting/")
		fc.puts = append(fc.puts, key)
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading PUT body: %v", err)
		}
		reply(w, string(raw))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	apiClient, err := unifi.New(context.Background(), &unifi.Config{
		BaseURL:  srv.URL,
		Username: "admin",
		Password: "admin",
	})
	if err != nil {
		t.Fatalf("creating client against fake controller: %v", err)
	}
	return &Client{ApiClient: apiClient, Site: "default"}
}

// TestSettingImportReadsEverySection: readSettings refreshes only the sections
// already in state, so an import read nothing and every section the
// configuration set (IGMP snooping, NTP, syslog, mgmt, ...) planned as an
// update rewriting the live values. An import must read every section the
// controller has, plan no changes for a configuration that matches it, and
// an unrelated update must write only the section the configuration sets.
func TestSettingImportReadsEverySection(t *testing.T) {
	ctx := context.Background()
	fc := &fakeSettingController{docs: fakeSettingDocuments}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	var identityResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)

	importResp := &fwresource.ImportStateResponse{
		State: tfsdk.State{Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil), Schema: s},
		Identity: &tfsdk.ResourceIdentity{
			Raw: tftypes.NewValue(
				identityResp.IdentitySchema.Type().TerraformType(ctx), nil,
			),
			Schema: identityResp.IdentitySchema,
		},
	}
	r.ImportState(ctx, fwresource.ImportStateRequest{ID: "default"}, importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", importResp.Diagnostics)
	}

	var imported settingResourceModel
	if d := importResp.State.Get(ctx, &imported); d.HasError() {
		t.Fatalf("reading imported state: %v", d)
	}
	for name, section := range map[string]types.Object{
		"ntp":           imported.Ntp,
		"mgmt":          imported.Mgmt,
		"igmp_snooping": imported.IgmpSnooping,
		"global_switch": imported.GlobalSwitch,
		"usg":           imported.USG,
		"dpi":           imported.Dpi,
		"syslog":        imported.Syslog,
	} {
		if section.IsNull() || section.IsUnknown() {
			t.Errorf("%s imported as %v, want the controller's document", name, section)
		}
	}
	for name, section := range map[string]types.Object{
		"country":      imported.Country,
		"connectivity": imported.Connectivity,
		"lcm":          imported.Lcm,
	} {
		if !section.IsNull() {
			t.Errorf("%s imported as %v, want null: the controller has no document", name, section)
		}
	}
	if !imported.Mgmt.IsNull() {
		var mgmt settingMgmtModel
		var ssh settingMgmtSSHModel
		if d := imported.Mgmt.As(ctx, &mgmt, basetypes.ObjectAsOptions{}); d.HasError() {
			t.Fatalf("mgmt: %v", d)
		}
		if d := mgmt.SSH.As(ctx, &ssh, basetypes.ObjectAsOptions{}); d.HasError() {
			t.Fatalf("mgmt.ssh: %v", d)
		}
		if !ssh.Enabled.Equal(types.BoolValue(true)) || ssh.Username.ValueString() != "admin" {
			t.Errorf("mgmt.ssh = %+v, want the controller's enabled admin login", ssh)
		}
		if !ssh.Password.IsNull() {
			t.Errorf("mgmt.ssh.password = %v, want null: only a hash is stored", ssh.Password)
		}
	}

	if t.Failed() {
		// The rest builds the configuration from the imported sections.
		return
	}

	// The refresh that follows the import reads the same values.
	readResp := &fwresource.ReadResponse{State: importResp.State, Identity: importResp.Identity}
	r.Read(ctx, fwresource.ReadRequest{
		State:    importResp.State,
		Identity: importResp.Identity,
	}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
	prior := readResp.State.Raw
	if diffs, _ := importResp.State.Raw.Diff(prior); len(diffs) != 0 {
		t.Errorf("refresh after import changed the state: %v", diffs)
	}

	// A configuration that sets some sections in full and others partially
	// plans no changes. Terraform core proposes the prior value for a section
	// the configuration leaves null, and the configuration value, with null
	// for every leaf it leaves out, for a section it sets.
	var priorAttrs map[string]tftypes.Value
	if err := prior.As(&priorAttrs); err != nil {
		t.Fatalf("prior as object: %v", err)
	}
	configAttrs := make(map[string]tftypes.Value, len(priorAttrs))
	for name, v := range priorAttrs {
		configAttrs[name] = tftypes.NewValue(v.Type(), nil)
	}
	for _, name := range []string{"ntp", "igmp_snooping", "global_switch", "usg"} {
		configAttrs[name] = priorAttrs[name]
	}
	var mgmtAttrs map[string]tftypes.Value
	if err := priorAttrs["mgmt"].As(&mgmtAttrs); err != nil {
		t.Fatalf("mgmt as object: %v", err)
	}
	partialMgmt := make(map[string]tftypes.Value, len(mgmtAttrs))
	for name, v := range mgmtAttrs {
		partialMgmt[name] = tftypes.NewValue(v.Type(), nil)
	}
	partialMgmt["ssh"] = mgmtAttrs["ssh"]
	configAttrs["mgmt"] = tftypes.NewValue(priorAttrs["mgmt"].Type(), partialMgmt)
	config := tftypes.NewValue(prior.Type(), configAttrs)
	proposed := func(configAttrs map[string]tftypes.Value) tftypes.Value {
		attrs := make(map[string]tftypes.Value, len(priorAttrs))
		for name, v := range priorAttrs {
			attrs[name] = v
			if !configAttrs[name].IsNull() {
				attrs[name] = configAttrs[name]
			}
		}
		return tftypes.NewValue(prior.Type(), attrs)
	}

	// dpi and syslog are left out although the controller holds values other
	// than their leaf defaults: the defaults must not reach an omitted section.
	planned := planResourceChange(t, "unifi_setting", s, prior, proposed(configAttrs), config)
	if diffs, err := prior.Diff(planned); err != nil || len(diffs) != 0 {
		t.Errorf("import of a matching configuration planned changes: %v (%v)", diffs, err)
	}

	// A configured section still gets the default of a leaf it leaves out.
	var dpiAttrs map[string]tftypes.Value
	if err := priorAttrs["dpi"].As(&dpiAttrs); err != nil {
		t.Fatalf("dpi as object: %v", err)
	}
	dpiAttrs["enabled"] = tftypes.NewValue(tftypes.Bool, nil)
	configAttrs["dpi"] = tftypes.NewValue(priorAttrs["dpi"].Type(), dpiAttrs)
	planned = planResourceChange(
		t, "unifi_setting", s, prior, proposed(configAttrs),
		tftypes.NewValue(prior.Type(), configAttrs),
	)
	var plannedAttrs, plannedDpi map[string]tftypes.Value
	if err := planned.As(&plannedAttrs); err != nil {
		t.Fatalf("planned as object: %v", err)
	}
	if err := plannedAttrs["dpi"].As(&plannedDpi); err != nil {
		t.Fatalf("planned dpi as object: %v", err)
	}
	var dpiEnabled bool
	if err := plannedDpi["enabled"].As(&dpiEnabled); err != nil || dpiEnabled {
		t.Errorf("dpi.enabled planned %v (%v), want the default false", plannedDpi["enabled"], err)
	}

	// A configured netconsole that leaves port out keeps the imported port
	// while another section changes; only its enabled default applies.
	var syslogAttrs, ncAttrs, ntpAttrs map[string]tftypes.Value
	if err := priorAttrs["syslog"].As(&syslogAttrs); err != nil {
		t.Fatalf("syslog as object: %v", err)
	}
	if err := syslogAttrs["netconsole"].As(&ncAttrs); err != nil {
		t.Fatalf("syslog.netconsole as object: %v", err)
	}
	partialNC := map[string]tftypes.Value{
		"enabled": tftypes.NewValue(tftypes.Bool, nil),
		"host":    ncAttrs["host"],
		"port":    tftypes.NewValue(tftypes.Number, nil),
	}
	partialSyslog := make(map[string]tftypes.Value, len(syslogAttrs))
	for name, v := range syslogAttrs {
		partialSyslog[name] = tftypes.NewValue(v.Type(), nil)
	}
	partialSyslog["enabled"] = syslogAttrs["enabled"]
	partialSyslog["ip"] = syslogAttrs["ip"]
	partialSyslog["netconsole"] = tftypes.NewValue(syslogAttrs["netconsole"].Type(), partialNC)
	configAttrs["syslog"] = tftypes.NewValue(priorAttrs["syslog"].Type(), partialSyslog)
	if err := priorAttrs["ntp"].As(&ntpAttrs); err != nil {
		t.Fatalf("ntp as object: %v", err)
	}
	ntpAttrs["ntp_server_2"] = tftypes.NewValue(tftypes.String, "time.google.com")
	configAttrs["ntp"] = tftypes.NewValue(priorAttrs["ntp"].Type(), ntpAttrs)
	planned = planResourceChange(
		t, "unifi_setting", s, prior, proposed(configAttrs),
		tftypes.NewValue(prior.Type(), configAttrs),
	)
	plannedAttrs = plannedObject(t, planned, "planned")
	plannedNC := plannedObject(
		t, plannedObject(t, plannedAttrs["syslog"], "syslog")["netconsole"], "syslog.netconsole",
	)
	if !plannedNC["port"].Equal(tftypes.NewValue(tftypes.Number, 6666)) {
		t.Errorf("syslog.netconsole.port planned %v, want the imported 6666", plannedNC["port"])
	}
	if !plannedNC["enabled"].Equal(tftypes.NewValue(tftypes.Bool, false)) {
		t.Errorf(
			"syslog.netconsole.enabled planned %v, want the default false",
			plannedNC["enabled"],
		)
	}

	// An unrelated change to the one section the configuration manages.
	var plan settingResourceModel
	if d := readResp.State.Get(ctx, &plan); d.HasError() {
		t.Fatalf("reading state: %v", d)
	}
	var ntp settingNtpModel
	if d := plan.Ntp.As(ctx, &ntp, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("ntp: %v", d)
	}
	ntp.NtpServer2 = types.StringValue("time.google.com")
	var d diag.Diagnostics
	plan.Ntp, d = types.ObjectValueFrom(ctx, ntpAttrTypes, ntp)
	if d.HasError() {
		t.Fatalf("ntp: %v", d)
	}
	onlyNtp := plan
	onlyNtp.AutoSpeedtest = types.ObjectNull(autoSpeedtestAttrTypes)
	onlyNtp.Country = types.ObjectNull(countryAttrTypes)
	onlyNtp.Dpi = types.ObjectNull(dpiAttrTypes)
	onlyNtp.Lcm = types.ObjectNull(lcmAttrTypes)
	onlyNtp.NetworkOpt = types.ObjectNull(networkOptimizationAttrTypes)
	onlyNtp.Syslog = types.ObjectNull(syslogAttrTypes)
	onlyNtp.Doh = types.ObjectNull(dohAttrTypes)
	onlyNtp.Ips = types.ObjectNull(ipsAttrTypes)
	onlyNtp.Mgmt = types.ObjectNull(mgmtAttrTypes)
	onlyNtp.Radius = types.ObjectNull(radiusAttrTypes)
	onlyNtp.USG = types.ObjectNull(usgAttrTypes)
	onlyNtp.IgmpSnooping = types.ObjectNull(igmpSnoopingAttrTypes)
	onlyNtp.Connectivity = types.ObjectNull(connectivityAttrTypes)
	onlyNtp.GlobalSwitch = types.ObjectNull(globalSwitchAttrTypes)

	planState := tfsdk.Plan{Schema: s}
	if d := planState.Set(ctx, &plan); d.HasError() {
		t.Fatalf("building plan: %v", d)
	}
	configPlan := tfsdk.Plan{Schema: s}
	if d := configPlan.Set(ctx, &onlyNtp); d.HasError() {
		t.Fatalf("building config: %v", d)
	}
	configState := tfsdk.Config(configPlan)
	updateResp := &fwresource.UpdateResponse{State: readResp.State, Identity: readResp.Identity}
	r.Update(ctx, fwresource.UpdateRequest{
		State:    readResp.State,
		Plan:     planState,
		Config:   configState,
		Identity: readResp.Identity,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics)
	}
	if strings.Join(fc.puts, ",") != "ntp" {
		t.Errorf("update wrote %v, want only the configured ntp section", fc.puts)
	}
}

// allSettingDocuments holds a document for every section the resource models.
var allSettingDocuments = map[string]string{
	"auto_speedtest": `{"key":"auto_speedtest","enabled":true,"cron_expr":"0 8 * * *"}`,
	"country":        `{"key":"country","code":840}`,
	"lcm": `{"key":"lcm","enabled":true,"brightness":80,"idle_timeout":300,"sync":true,
		"touch_event":true}`,
	"network_optimization": `{"key":"network_optimization","enabled":true}`,
	"doh": `{"key":"doh","state":"off","server_names":["cloudflare"],
		"custom_servers":[]}`,
	"ips": `{"key":"ips","ips_mode":"disabled","honeypot_enabled":false,
		"restrict_torrents":false,"content_filtering_blocking_page_enabled":true,
		"memory_optimized":true,"advanced_filtering_preference":"manual",
		"enabled_categories":[],"enabled_networks":["lan-a"],"honeypot":[]}`,
	"radius": `{"key":"radius","accounting_enabled":false,"acct_port":1813,"auth_port":1812,
		"interim_update_interval":3600,"x_secret":"radius-secret"}`,
	"connectivity":  `{"key":"connectivity","enabled":false,"uplink_type":"gateway"}`,
	"dpi":           fakeSettingDocuments["dpi"],
	"rsyslogd":      fakeSettingDocuments["rsyslogd"],
	"ntp":           fakeSettingDocuments["ntp"],
	"mgmt":          fakeSettingDocuments["mgmt"],
	"igmp_snooping": fakeSettingDocuments["igmp_snooping"],
	"global_switch": fakeSettingDocuments["global_switch"],
	"usg":           fakeSettingDocuments["usg"],
}

// importSettings runs ImportState for the site "default" against r, by
// identity when byIdentity is set and by import id otherwise.
func importSettings(
	ctx context.Context,
	t *testing.T,
	r *settingResource,
	byIdentity bool,
) *fwresource.ImportStateResponse {
	t.Helper()
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	var identityResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)
	newIdentity := func() *tfsdk.ResourceIdentity {
		return &tfsdk.ResourceIdentity{
			Raw: tftypes.NewValue(
				identityResp.IdentitySchema.Type().TerraformType(ctx), nil,
			),
			Schema: identityResp.IdentitySchema,
		}
	}
	resp := &fwresource.ImportStateResponse{
		State: tfsdk.State{
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			Schema: schemaResp.Schema,
		},
		Identity: newIdentity(),
	}
	req := fwresource.ImportStateRequest{ID: "default"}
	if byIdentity {
		identity := newIdentity()
		if d := identity.Set(
			ctx,
			settingIdentityModel{Site: types.StringValue("default")},
		); d.HasError() {
			t.Fatalf("building import identity: %v", d)
		}
		req = fwresource.ImportStateRequest{Identity: identity}
	}
	r.ImportState(ctx, req, resp)
	return resp
}

// An import block keyed by resource identity (Terraform 1.12+) reads every
// section the controller has, exactly like an import by id.
func TestSettingImportByIdentityReadsEverySection(t *testing.T) {
	ctx := context.Background()
	fc := &fakeSettingController{docs: allSettingDocuments}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	resp := importSettings(ctx, t, r, true)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", resp.Diagnostics)
	}
	var imported settingResourceModel
	if d := resp.State.Get(ctx, &imported); d.HasError() {
		t.Fatalf("reading imported state: %v", d)
	}
	for name, section := range map[string]types.Object{
		"auto_speedtest":       imported.AutoSpeedtest,
		"country":              imported.Country,
		"dpi":                  imported.Dpi,
		"lcm":                  imported.Lcm,
		"network_optimization": imported.NetworkOpt,
		"ntp":                  imported.Ntp,
		"syslog":               imported.Syslog,
		"doh":                  imported.Doh,
		"ips":                  imported.Ips,
		"mgmt":                 imported.Mgmt,
		"radius":               imported.Radius,
		"usg":                  imported.USG,
		"igmp_snooping":        imported.IgmpSnooping,
		"connectivity":         imported.Connectivity,
		"global_switch":        imported.GlobalSwitch,
	} {
		if section.IsNull() || section.IsUnknown() {
			t.Errorf("%s imported as %v, want the controller's document", name, section)
		}
	}
	if !imported.Radius.IsNull() {
		var radius settingRadiusModel
		if d := imported.Radius.As(ctx, &radius, basetypes.ObjectAsOptions{}); d.HasError() {
			t.Fatalf("radius: %v", d)
		}
		if !radius.Secret.IsNull() {
			t.Errorf("radius.secret imported as %v, want null: never import the secret", radius.Secret)
		}
	}
	if len(fc.puts) != 0 {
		t.Errorf("import wrote %v, want no writes", fc.puts)
	}
}

// A usg document without dns_verification imports dns_verification as an
// empty object, and the refresh that follows reads it the same way.
func TestSettingImportWithoutUSGDNSVerification(t *testing.T) {
	ctx := context.Background()
	docs := maps.Clone(fakeSettingDocuments)
	docs["usg"] = `{"key":"usg","broadcast_ping":false,"ftp_module":true,
		"mss_clamp":"auto","timeout_setting_preference":"auto","upnp_enabled":false}`
	fc := &fakeSettingController{docs: docs}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	resp := importSettings(ctx, t, r, false)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", resp.Diagnostics)
	}
	var imported settingResourceModel
	if d := resp.State.Get(ctx, &imported); d.HasError() {
		t.Fatalf("reading imported state: %v", d)
	}
	var usg settingUSGModel
	if d := imported.USG.As(ctx, &usg, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("usg: %v", d)
	}
	var dns dnsVerificationModel
	if d := usg.DNSVerification.As(ctx, &dns, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("usg.dns_verification: %v", d)
	}
	if !dns.Domain.Equal(types.StringValue("")) ||
		!dns.SettingPreference.Equal(types.StringValue("")) {
		t.Errorf("usg.dns_verification = %+v, want known empty values", dns)
	}

	readResp := &fwresource.ReadResponse{State: resp.State, Identity: resp.Identity}
	r.Read(ctx, fwresource.ReadRequest{State: resp.State, Identity: resp.Identity}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
	if diffs, _ := resp.State.Raw.Diff(readResp.State.Raw); len(diffs) != 0 {
		t.Errorf("refresh after import changed the state: %v", diffs)
	}
}

// An import fails cleanly, writing nothing, when the controller cannot list
// its settings or cannot return a document it listed.
func TestSettingImportReportsControllerErrors(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fc   *fakeSettingController
		want string
	}{
		{
			name: "settings cannot be listed",
			fc:   &fakeSettingController{docs: fakeSettingDocuments, failList: true},
			want: "Error Listing Settings",
		},
		{
			name: "a listed document cannot be read",
			fc: &fakeSettingController{
				docs:       fakeSettingDocuments,
				unreadable: map[string]bool{"ntp": true},
			},
			want: "Error Reading NTP Setting",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &settingResource{client: newSettingFakeControllerClient(t, tc.fc)}
			resp := importSettings(ctx, t, r, false)
			if !resp.Diagnostics.HasError() {
				t.Fatal("ImportState succeeded, want an error")
			}
			if got := resp.Diagnostics.Errors()[0].Summary(); got != tc.want {
				t.Errorf("error summary = %q, want %q", got, tc.want)
			}
			if len(tc.fc.puts) != 0 {
				t.Errorf("import wrote %v, want no writes", tc.fc.puts)
			}
		})
	}
}

// readImportedSettings leaves state alone after an earlier import error, and
// reports a state it cannot decode before it reaches the controller.
func TestReadImportedSettingsStopsBeforeTheController(t *testing.T) {
	ctx := context.Background()
	// No client: reaching the controller would panic.
	r := &settingResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)

	t.Run("earlier error", func(t *testing.T) {
		state := tfsdk.State{
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
			Schema: schemaResp.Schema,
		}
		var diags diag.Diagnostics
		diags.AddError("earlier", "import id was invalid")
		r.readImportedSettings(ctx, "default", &state, &diags)
		if len(diags.Errors()) != 1 {
			t.Errorf("diagnostics = %v, want only the earlier error", diags)
		}
		if !state.Raw.IsNull() {
			t.Errorf("state = %v, want it left null", state.Raw)
		}
	})

	t.Run("undecodable state", func(t *testing.T) {
		reduced := schemaResp.Schema
		reduced.Attributes = maps.Clone(schemaResp.Schema.Attributes)
		delete(reduced.Attributes, "ntp")
		state := tfsdk.State{
			Raw:    tftypes.NewValue(reduced.Type().TerraformType(ctx), nil),
			Schema: reduced,
		}
		var diags diag.Diagnostics
		r.readImportedSettings(ctx, "default", &state, &diags)
		if !diags.HasError() {
			t.Error("readImportedSettings accepted a state without the ntp section")
		}
	})
}

// After an import every section is in the plan. When the configuration sets
// all of them, an update writes each one; and an update whose configuration
// cannot be decoded writes nothing.
func TestSettingUpdateWritesEveryConfiguredSection(t *testing.T) {
	ctx := context.Background()
	fc := &fakeSettingController{docs: allSettingDocuments}
	r := &settingResource{client: newSettingFakeControllerClient(t, fc)}

	importResp := importSettings(ctx, t, r, false)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", importResp.Diagnostics)
	}
	state := importResp.State
	var plan settingResourceModel
	if d := state.Get(ctx, &plan); d.HasError() {
		t.Fatalf("reading imported state: %v", d)
	}
	var ntp settingNtpModel
	if d := plan.Ntp.As(ctx, &ntp, basetypes.ObjectAsOptions{}); d.HasError() {
		t.Fatalf("ntp: %v", d)
	}
	ntp.NtpServer2 = types.StringValue("time.google.com")
	var d diag.Diagnostics
	plan.Ntp, d = types.ObjectValueFrom(ctx, ntpAttrTypes, ntp)
	if d.HasError() {
		t.Fatalf("ntp: %v", d)
	}
	planned := tfsdk.Plan{Schema: state.Schema}
	if d := planned.Set(ctx, &plan); d.HasError() {
		t.Fatalf("building plan: %v", d)
	}

	t.Run("undecodable configuration", func(t *testing.T) {
		fc.puts = nil
		resp := &fwresource.UpdateResponse{State: state, Identity: importResp.Identity}
		r.Update(ctx, fwresource.UpdateRequest{
			State:    state,
			Plan:     planned,
			Config:   tfsdk.Config{Schema: state.Schema},
			Identity: importResp.Identity,
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("Update accepted a configuration it cannot decode")
		}
		if len(fc.puts) != 0 {
			t.Errorf("update wrote %v, want no writes", fc.puts)
		}
	})

	t.Run("every section configured", func(t *testing.T) {
		fc.puts = nil
		resp := &fwresource.UpdateResponse{State: state, Identity: importResp.Identity}
		r.Update(ctx, fwresource.UpdateRequest{
			State:    state,
			Plan:     planned,
			Config:   tfsdk.Config(planned),
			Identity: importResp.Identity,
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		written := map[string]bool{}
		for _, key := range fc.puts {
			written[key] = true
		}
		for _, key := range []string{
			"auto_speedtest", "country", "dpi", "lcm", "network_optimization", "ntp",
			"rsyslogd", "doh", "ips", "mgmt", "radius", "usg", "igmp_snooping",
			"connectivity", "global_switch",
		} {
			if !written[key] {
				t.Errorf("update did not write the configured %s section (wrote %v)", key, fc.puts)
			}
		}
	})
}

// keepPriorUnconfigured skips an attribute the configuration or the prior
// state does not carry, and leaves non-object values and unknown or null
// objects untouched.
func TestKeepPriorUnconfiguredSkipsMissingAttributes(t *testing.T) {
	ctx := context.Background()
	full := map[string]attr.Type{"a": types.StringType, "b": types.StringType}
	plan := types.ObjectValueMust(full, map[string]attr.Value{
		"a": types.StringUnknown(),
		"b": types.StringUnknown(),
	})
	prior := types.ObjectValueMust(full, map[string]attr.Value{
		"a": types.StringValue("prior-a"),
		"b": types.StringValue("prior-b"),
	})
	// The configuration carries only "a".
	config := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType},
		map[string]attr.Value{"a": types.StringNull()},
	)

	got, d := keepPriorUnconfigured(ctx, plan, config, prior)
	if d.HasError() {
		t.Fatalf("keepPriorUnconfigured: %v", d)
	}
	attrs := got.Attributes()
	if !attrs["a"].Equal(types.StringValue("prior-a")) {
		t.Errorf("a = %v, want the prior value", attrs["a"])
	}
	if !attrs["b"].IsUnknown() {
		t.Errorf("b = %v, want it left unknown: the configuration does not carry it", attrs["b"])
	}

	for name, tc := range map[string]struct{ plan, config, prior types.Object }{
		"unknown plan":  {types.ObjectUnknown(full), config, prior},
		"null config":   {plan, types.ObjectNull(full), prior},
		"unknown prior": {plan, config, types.ObjectUnknown(full)},
	} {
		got, d := keepPriorUnconfigured(ctx, tc.plan, tc.config, tc.prior)
		if d.HasError() || !got.Equal(tc.plan) {
			t.Errorf("%s: got %v (%v), want the plan unchanged", name, got, d)
		}
	}
}

// failingStringType is a string type whose values cannot be built.
type failingStringType struct {
	basetypes.StringType
}

func (failingStringType) ValueFromTerraform(
	context.Context,
	tftypes.Value,
) (attr.Value, error) {
	return nil, errors.New("cannot build a value")
}

// knownObject reports a type whose zero value cannot be built rather than
// returning a partial seed.
func TestKnownObjectReportsUnbuildableTypes(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"ok":     types.StringType,
		"broken": failingStringType{},
	}
	got, d := knownObject(context.Background(), attrTypes)
	if !d.HasError() {
		t.Fatal("knownObject built a seed for a type it cannot build")
	}
	if !got.IsNull() {
		t.Errorf("seed = %v, want null on error", got)
	}
}

// settingModifyPlanRequest builds a ModifyPlan request and response from raw
// values, each paired with its own schema, so the defensive error branches
// can be driven with a mismatched schema.
func settingModifyPlanRequest(
	planSchema, configSchema, stateSchema schema.Schema,
	plan, config, state tftypes.Value,
) (fwresource.ModifyPlanRequest, *fwresource.ModifyPlanResponse) {
	req := fwresource.ModifyPlanRequest{
		Plan:   tfsdk.Plan{Schema: planSchema, Raw: plan},
		Config: tfsdk.Config{Schema: configSchema, Raw: config},
		State:  tfsdk.State{Schema: stateSchema, Raw: state},
	}
	return req, &fwresource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: planSchema, Raw: plan},
	}
}

// withoutAttribute returns s and raw (an object of s) without attribute name.
func withoutAttribute(
	ctx context.Context,
	t *testing.T,
	s schema.Schema,
	raw tftypes.Value,
	name string,
) (schema.Schema, tftypes.Value) {
	t.Helper()
	reduced := s
	reduced.Attributes = maps.Clone(s.Attributes)
	delete(reduced.Attributes, name)
	var attrs map[string]tftypes.Value
	if err := raw.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	attrs = maps.Clone(attrs)
	delete(attrs, name)
	return reduced, tftypes.NewValue(reduced.Type().TerraformType(ctx), attrs)
}

// ModifyPlan stops with a diagnostic, rather than planning from half-read
// values, when a request does not match the resource schema.
func TestSettingModifyPlanReportsMismatchedRequests(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	schemaType := s.Type().TerraformType(ctx)

	f := planFixture{ctx: ctx, t: t}
	prior, _ := f.object(s, "")
	var priorAttrs map[string]tftypes.Value
	if err := prior.As(&priorAttrs); err != nil {
		t.Fatalf("prior as object: %v", err)
	}
	// A configuration that sets auto_speedtest but leaves enabled out, so
	// its default applies.
	configAttrs := make(map[string]tftypes.Value, len(priorAttrs))
	for name, v := range priorAttrs {
		configAttrs[name] = tftypes.NewValue(v.Type(), nil)
	}
	speedtestType := priorAttrs["auto_speedtest"].Type()
	configAttrs["auto_speedtest"] = tftypes.NewValue(speedtestType, map[string]tftypes.Value{
		"enabled":   tftypes.NewValue(tftypes.Bool, nil),
		"cron_expr": tftypes.NewValue(tftypes.String, "0 * * * *"),
	})
	config := tftypes.NewValue(schemaType, configAttrs)
	nullState := tftypes.NewValue(schemaType, nil)

	t.Run("prior state without a section", func(t *testing.T) {
		stateSchema, state := withoutAttribute(ctx, t, s, prior, "ntp")
		req, resp := settingModifyPlanRequest(s, s, stateSchema, prior, config, state)
		r.ModifyPlan(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ModifyPlan accepted a prior state without the ntp section")
		}
	})

	t.Run("prior section of another shape", func(t *testing.T) {
		// The prior state's ntp section holds a number where the plan holds a
		// string, so its value cannot be carried into the plan.
		stateSchema := s
		stateSchema.Attributes = maps.Clone(s.Attributes)
		ntpAttr, ok := s.Attributes["ntp"].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("ntp is %T, want schema.SingleNestedAttribute", s.Attributes["ntp"])
		}
		ntpAttr.Attributes = maps.Clone(ntpAttr.Attributes)
		ntpAttr.Attributes["ntp_server_1"] = schema.Int64Attribute{Optional: true, Computed: true}
		stateSchema.Attributes["ntp"] = ntpAttr
		stateAttrs := maps.Clone(priorAttrs)
		var ntp map[string]tftypes.Value
		if err := priorAttrs["ntp"].As(&ntp); err != nil {
			t.Fatalf("ntp as object: %v", err)
		}
		ntp = maps.Clone(ntp)
		ntp["ntp_server_1"] = tftypes.NewValue(tftypes.Number, 1)
		stateAttrs["ntp"] = tftypes.NewValue(
			stateSchema.Attributes["ntp"].GetType().TerraformType(ctx), ntp,
		)
		state := tftypes.NewValue(stateSchema.Type().TerraformType(ctx), stateAttrs)

		// The plan leaves ntp_server_1 unknown and the configuration sets ntp
		// without it, so the prior value is the one that would be carried.
		planAttrs := maps.Clone(priorAttrs)
		planNtp := maps.Clone(ntp)
		planNtp["ntp_server_1"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		planAttrs["ntp"] = tftypes.NewValue(priorAttrs["ntp"].Type(), planNtp)
		plan := tftypes.NewValue(schemaType, planAttrs)
		cfgAttrs := maps.Clone(configAttrs)
		cfgNtp := maps.Clone(planNtp)
		cfgNtp["ntp_server_1"] = tftypes.NewValue(tftypes.String, nil)
		cfgAttrs["ntp"] = tftypes.NewValue(priorAttrs["ntp"].Type(), cfgNtp)

		req, resp := settingModifyPlanRequest(
			s, s, stateSchema, plan, tftypes.NewValue(schemaType, cfgAttrs), state,
		)
		r.ModifyPlan(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ModifyPlan carried a prior value of another type into the plan")
		}
	})

	t.Run("configuration without a defaulted section", func(t *testing.T) {
		configSchema, reducedConfig := withoutAttribute(ctx, t, s, config, "auto_speedtest")
		req, resp := settingModifyPlanRequest(
			s, configSchema, s, prior, reducedConfig, nullState,
		)
		r.ModifyPlan(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ModifyPlan accepted a configuration without the auto_speedtest section")
		}
	})

	t.Run("plan without the last defaulted object", func(t *testing.T) {
		// The configuration sets syslog.this_controller without its enabled
		// leaf, and the plan's syslog has no this_controller to default.
		planSchema := s
		planSchema.Attributes = maps.Clone(s.Attributes)
		syslogAttr, ok := s.Attributes["syslog"].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("syslog is %T, want schema.SingleNestedAttribute", s.Attributes["syslog"])
		}
		syslogAttr.Attributes = maps.Clone(syslogAttr.Attributes)
		delete(syslogAttr.Attributes, "this_controller")
		planSchema.Attributes["syslog"] = syslogAttr
		var syslog map[string]tftypes.Value
		if err := priorAttrs["syslog"].As(&syslog); err != nil {
			t.Fatalf("syslog as object: %v", err)
		}
		planSyslog := maps.Clone(syslog)
		delete(planSyslog, "this_controller")
		planAttrs := maps.Clone(priorAttrs)
		planAttrs["syslog"] = tftypes.NewValue(
			syslogAttr.GetType().TerraformType(ctx), planSyslog,
		)
		plan := tftypes.NewValue(planSchema.Type().TerraformType(ctx), planAttrs)

		cfgSyslog := maps.Clone(syslog)
		var thisController map[string]tftypes.Value
		if err := syslog["this_controller"].As(&thisController); err != nil {
			t.Fatalf("this_controller as object: %v", err)
		}
		thisController = maps.Clone(thisController)
		thisController["enabled"] = tftypes.NewValue(tftypes.Bool, nil)
		cfgSyslog["this_controller"] = tftypes.NewValue(
			syslog["this_controller"].Type(), thisController,
		)
		cfgAttrs := maps.Clone(configAttrs)
		cfgAttrs["syslog"] = tftypes.NewValue(priorAttrs["syslog"].Type(), cfgSyslog)

		req, resp := settingModifyPlanRequest(
			planSchema, s, s, plan, tftypes.NewValue(schemaType, cfgAttrs), nullState,
		)
		r.ModifyPlan(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ModifyPlan applied a default to a plan without syslog.this_controller")
		}
	})

	t.Run("plan without a defaulted section", func(t *testing.T) {
		planSchema, plan := withoutAttribute(ctx, t, s, prior, "auto_speedtest")
		req, resp := settingModifyPlanRequest(planSchema, s, s, plan, config, nullState)
		r.ModifyPlan(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ModifyPlan applied a default to a plan without the auto_speedtest section")
		}
	})
}
