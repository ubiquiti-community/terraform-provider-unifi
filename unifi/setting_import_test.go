package unifi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
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

// newSettingFakeControllerClient serves fakeSettingDocuments from a minimal
// fake controller (old-style API: 302 on /, cookie login at /api/login) and
// records the setting key of every PUT to it.
func newSettingFakeControllerClient(t *testing.T, puts *[]string) *Client {
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
		keys := make([]string, 0, len(fakeSettingDocuments))
		for key := range fakeSettingDocuments {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		docs := make([]string, 0, len(keys))
		for _, key := range keys {
			docs = append(docs, fakeSettingDocuments[key])
		}
		reply(w, docs...)
	})
	mux.HandleFunc("/api/s/default/get/setting/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/s/default/get/setting/")
		if doc, ok := fakeSettingDocuments[key]; ok {
			reply(w, doc)
			return
		}
		reply(w)
	})
	mux.HandleFunc("/api/s/default/set/setting/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/s/default/set/setting/")
		*puts = append(*puts, key)
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
	var puts []string
	r := &settingResource{client: newSettingFakeControllerClient(t, &puts)}

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
		t.Errorf("syslog.netconsole.enabled planned %v, want the default false", plannedNC["enabled"])
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
	if strings.Join(puts, ",") != "ntp" {
		t.Errorf("update wrote %v, want only the configured ntp section", puts)
	}
}
