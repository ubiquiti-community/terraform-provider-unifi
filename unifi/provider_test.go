package unifi

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/docker/compose/v2/pkg/api"
	fwaction "github.com/hashicorp/terraform-plugin-framework/action"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwephemeral "github.com/hashicorp/terraform-plugin-framework/ephemeral"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New()),
}

var testAccProtoV6ProviderFactories = providerFactories

var _ *unifi.ApiClient

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	// UNIFI_SKIP_CONTAINER bypasses docker-compose and uses pre-set UNIFI_* env vars.
	if os.Getenv("UNIFI_SKIP_CONTAINER") != "" {
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

type logConsumer struct {
	StdOut bool

	ctx context.Context
}

func (l *logConsumer) Accept(log testcontainers.Log) {
	switch log.LogType {
	case testcontainers.StdoutLog:
		tflog.Info(l.ctx, string(log.Content))
	case testcontainers.StderrLog:
		tflog.Error(l.ctx, string(log.Content))
	}
}

func runAcceptanceTests(m *testing.M) int {
	// Disable Ryuk reaper to avoid connection issues in local development
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(
		tflogtest.RootLogger(context.Background(), os.Stdout),
	)
	defer cancel()

	logger := NewLogger(ctx)

	_ = compose.WithLogger(logger)

	dc, err := compose.NewDockerCompose("../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	// Don't wait for health check in compose.Up - we'll do our own waiting with waitForUniFiAPI
	// The health check has a 90s start_period which can cause timeouts in testcontainers
	if err = dc.WithOsEnv().
		Up(ctx, compose.Wait(true), compose.WithRecreate(api.RecreateDiverged)); err != nil {
		panic(err)
	}

	container, err := dc.ServiceContainer(ctx, "unifi")
	if err != nil {
		panic(err)
	}

	lc := &logConsumer{StdOut: os.Getenv("UNIFI_STDOUT") == ""}

	testcontainers.WithLogConsumers(lc)

	// Get the host that the container is accessible from
	host, err := container.Host(ctx)
	if err != nil {
		panic(err)
	}

	// Get the mapped port for 8443
	mappedPort, err := container.MappedPort(ctx, "8443/tcp")
	if err != nil {
		panic(err)
	}

	endpoint := fmt.Sprintf("https://%s:%s", host, mappedPort.Port())
	logger.Printf("UniFi controller endpoint: %s", endpoint)

	const user = "admin"
	const password = "admin"

	if err = os.Setenv("UNIFI_USERNAME", user); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_PASSWORD", password); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_INSECURE", "true"); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_API", endpoint); err != nil {
		panic(err)
	}

	if err = os.Setenv("UNIFI_API_KEY", ""); err != nil {
		panic(err)
	}

	defer func() {
		logger.Printf("RUNNING TEAR DOWN")
		if err := dc.Down(
			context.Background(),
			compose.RemoveOrphans(true),
			compose.RemoveImagesLocal,
		); err != nil {
			panic(err)
		}
	}()

	if _, err := waitForUniFiAPI(ctx, logger, endpoint, user, password); err != nil {
		panic(err)
	}

	return m.Run()
}

func preCheck(t *testing.T) {
	variables := []string{
		"UNIFI_USERNAME",
		"UNIFI_PASSWORD",
		"UNIFI_API",
	}

	for _, variable := range variables {
		value := os.Getenv(variable)
		if value == "" {
			t.Fatalf("`%s` must be set for acceptance tests!", variable)
		}
	}
}

// contains checks if a string contains a substring (helper to avoid importing strings).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// waitForUniFiAPI waits for the UniFi API to be ready and accepting JSON requests
// Docker health check ensures the container is ready, this validates full API initialization.
func waitForUniFiAPI(
	ctx context.Context,
	logger *UnifiLogger,
	endpoint, user, password string,
) (client *unifi.ApiClient, err error) {
	maxRetries := 60
	retryDelay := 3 * time.Second

	logger.Printf(
		"Waiting for UniFi API to be ready (max %d attempts, %v between attempts)...",
		maxRetries,
		retryDelay,
	)

	var loginSuccessful bool
	for i := range maxRetries {
		// Step 1: Try to login
		client, err = unifi.New(ctx, &unifi.Config{
			BaseURL:        endpoint,
			Username:       user,
			Password:       password,
			AllowInsecure:  true,
			TimeoutSeconds: util.Ptr(30),
			CloudConnector: false,
		})
		if err != nil {
			errMsg := err.Error()
			if i < maxRetries-1 {
				if (i+1)%10 == 0 {
					logger.Printf(
						"Still waiting for login... (attempt %d/%d): %v",
						i+1,
						maxRetries,
						errMsg,
					)
				}
				time.Sleep(retryDelay)
				continue
			}

			return nil, fmt.Errorf(
				"UniFi API login did not succeed after %d attempts (waited %v): %w",
				maxRetries,
				time.Duration(maxRetries)*retryDelay,
				err,
			)
		}

		if !loginSuccessful {
			logger.Printf("✓ Login successful after %d attempts", i+1)
			loginSuccessful = true
		}

		// Step 2: Verify sites are initialized
		sites, err := client.ListSites(ctx)
		if err != nil {

			// Not found errors are expected during initialization
			errStr := err.Error()
			if contains(errStr, "not found") || contains(errStr, "NotFound") ||
				contains(errStr, "404") {
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						logger.Printf(
							"Sites not initialized yet... (attempt %d/%d)",
							i+1,
							maxRetries,
						)
					}
					time.Sleep(retryDelay)
					continue
				}
			} else {
				// Unexpected error
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						logger.Printf(
							"Sites endpoint error... (attempt %d/%d): %v",
							i+1,
							maxRetries,
							err,
						)
					}
					time.Sleep(retryDelay)
					continue
				}

				return nil, fmt.Errorf(
					"UniFi API sites not ready after %d attempts: %w",
					maxRetries,
					err,
				)
			}
		}

		if len(sites) == 0 {
			if i < maxRetries-1 {
				if (i+1)%10 == 0 {
					logger.Printf(
						"No sites found yet... (attempt %d/%d)",
						i+1,
						maxRetries,
					)
				}
				time.Sleep(retryDelay)
				continue
			}

			return nil, fmt.Errorf("no sites available after %d attempts", maxRetries)
		}

		// Step 3: Verify we can list devices (API fully operational)
		if devices, err := client.ListDevice(ctx, "default"); err != nil {
			// This is acceptable - there may be no devices yet
			// But we want to verify the endpoint is responsive
			errStr := err.Error()
			if !contains(errStr, "404") && !contains(errStr, "NotFound") &&
				!contains(errStr, "not found") {
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						logger.Printf(
							"Devices endpoint not ready... (attempt %d/%d): %v",
							i+1,
							maxRetries,
							err,
						)
					}
					time.Sleep(retryDelay)
					continue
				}

				return nil, fmt.Errorf(
					"device endpoint not operational after %d attempts: %w",
					maxRetries,
					err,
				)
			}
		} else {
			for _, dev := range devices {
				if !dev.Adopted {
					adoptErr := client.AdoptDevice(ctx, "default", dev.MAC)
					if adoptErr != nil {
						logger.Printf(
							"Failed to adopt device %s: %v, retrying...",
							dev.MAC,
							adoptErr,
						)
						// Retry adoption once after a brief delay
						time.Sleep(2 * time.Second)
						adoptErr = client.AdoptDevice(ctx, "default", dev.MAC)
						if adoptErr != nil {
							logger.Printf(
								"Failed to adopt device %s after retry: %v",
								dev.MAC,
								adoptErr,
							)
						} else {
							logger.Printf("Successfully adopted device %s on retry", dev.MAC)
						}
					} else {
						logger.Printf("Successfully adopted device %s", dev.MAC)
					}
				}
			}
		}

		// Step 4: Verify networks are initialized
		networks, err := client.ListNetwork(ctx, "default")
		if err != nil {
			// Not found errors are expected during initialization
			errStr := err.Error()
			if contains(errStr, "not found") || contains(errStr, "NotFound") ||
				contains(errStr, "404") {
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						logger.Printf(
							"Networks not initialized yet... (attempt %d/%d)",
							i+1,
							maxRetries,
						)
					}
					time.Sleep(retryDelay)
					continue
				}
			} else {
				// Unexpected error
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						logger.Printf(
							"Networks endpoint error... (attempt %d/%d): %v",
							i+1,
							maxRetries,
							err,
						)
					}
					time.Sleep(retryDelay)
					continue
				}

				return nil, fmt.Errorf(
					"UniFi API networks not ready after %d attempts: %w",
					maxRetries,
					err,
				)
			}
		}

		if len(networks) == 0 {
			if i < maxRetries-1 {
				if (i+1)%10 == 0 {
					logger.Printf(
						"No networks found yet... (attempt %d/%d)",
						i+1,
						maxRetries,
					)
				}
				time.Sleep(retryDelay)
				continue
			}

			return nil, fmt.Errorf("no networks available after %d attempts", maxRetries)
		}

		// Step 5: Ensure a default WAN network exists
		hasWAN := false
		for _, n := range networks {
			if n.Purpose == unifi.PurposeWAN && n.WANNetworkGroup != nil &&
				*n.WANNetworkGroup == "WAN" {
				hasWAN = true
				break
			}
		}
		if !hasWAN {
			logger.Printf("No default WAN network found, creating \"Internet 1\"...")
			_, createErr := client.CreateNetwork(ctx, "default", &unifi.Network{
				Name:            util.Ptr("Internet 1"),
				Purpose:         unifi.PurposeWAN,
				WANNetworkGroup: util.Ptr("WAN"),
				WANType:         util.Ptr("dhcp"),
			})
			if createErr != nil {
				logger.Printf("Failed to create default WAN network: %v", createErr)
			} else {
				logger.Printf("✓ Created default WAN network \"Internet 1\"")
			}
		}

		logger.Printf(
			"✓ UniFi API fully ready (login + %d sites + devices endpoint) after %d attempts",
			len(sites),
			i+1,
		)
		return client, nil
	}

	return nil, fmt.Errorf("UniFi API did not become ready after %d attempts", maxRetries)
}

func TestClient_GetSiteName(t *testing.T) {
	tests := []struct {
		name string
		c    *Client
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.GetSiteName(); got != tt.want {
				t.Errorf("Client.GetSiteName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want fwprovider.Provider
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifiProvider_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwprovider.MetadataRequest
		resp *fwprovider.MetadataResponse
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.p.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_unifiProvider_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwprovider.SchemaRequest
		resp *fwprovider.SchemaResponse
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.p.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_unifiProvider_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwprovider.ConfigureRequest
		resp *fwprovider.ConfigureResponse
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.p.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_unifiProvider_Resources(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
		want []func() fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Resources(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unifiProvider.Resources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifiProvider_DataSources(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
		want []func() fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.DataSources(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unifiProvider.DataSources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifiProvider_EphemeralResources(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
		want []func() fwephemeral.EphemeralResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.EphemeralResources(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unifiProvider.EphemeralResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifiProvider_Actions(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
		want []func() fwaction.Action
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Actions(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unifiProvider.Actions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_unifiProvider_ListResources(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name string
		p    *unifiProvider
		args args
		want []func() fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.ListResources(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unifiProvider.ListResources() = %v, want %v", got, tt.want)
			}
		})
	}
}
