package unifi

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

var providerFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"unifi": providerserver.NewProtocol6WithError(New()),
}

var testAccProtoV6ProviderFactories = providerFactories

var testClient *unifi.Client

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" {
		// short circuit non acceptance test runs
		os.Exit(m.Run())
	}

	os.Exit(runAcceptanceTests(m))
}

type logConsumer struct {
	StdOut bool
}

func (l *logConsumer) Accept(log testcontainers.Log) {
	if log.LogType == testcontainers.StdoutLog && l.StdOut {
		testcontainers.Logger.Printf("%s", log.Content)
	}
	if log.LogType == testcontainers.StderrLog {
		testcontainers.Logger.Printf("%s", log.Content)
	}
}

func runAcceptanceTests(m *testing.M) int {
	// Disable Ryuk reaper to avoid connection issues in local development
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic(err)
	}

	_ = compose.WithLogger(log.Default())

	dc, err := compose.NewDockerCompose("../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Don't wait for health check in compose.Up - we'll do our own waiting with waitForUniFiAPI
	// The health check has a 90s start_period which can cause timeouts in testcontainers
	if err = dc.WithOsEnv().Up(ctx, compose.Wait(true), compose.WithRecreate(api.RecreateDiverged)); err != nil {
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
	log.Printf("UniFi controller endpoint: %s", endpoint)

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
		log.Print("RUNNING TEAR DOWN")
		if err := dc.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal); err != nil {
			panic(err)
		}
	}()

	testClient = &unifi.Client{}
	httpClient := retryablehttp.NewClient()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	httpClient.HTTPClient.Transport = transport
	jar, _ := cookiejar.New(nil)
	httpClient.HTTPClient.Jar = jar
	if err = testClient.SetHTTPClient(httpClient); err != nil {
		panic(err)
	}
	if err = testClient.SetBaseURL(endpoint); err != nil {
		panic(err)
	}
	if err = waitForUniFiAPI(ctx, testClient, user, password); err != nil {
		panic(err)
	}

		return m.Run()
}

func importStep(name string, ignore ...string) resource.TestStep {
	step := resource.TestStep{
		ResourceName:      name,
		ImportState:       true,
		ImportStateVerify: true,
	}

	if len(ignore) > 0 {
		step.ImportStateVerifyIgnore = ignore
	}

	return step
}

func siteAndIDImportStateIDFunc(resourceName string) func(*terraform.State) (string, error) {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}
		networkID := rs.Primary.Attributes["id"]
		site := rs.Primary.Attributes["site"]
		return site + ":" + networkID, nil
	}
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

const (
	vlanMin = 2
	vlanMax = 4095
)

var (
	network = &net.IPNet{
		IP:   net.IPv4(10, 0, 0, 0).To4(),
		Mask: net.IPv4Mask(255, 0, 0, 0),
	}

	vlanLock sync.Mutex
	vlanNext = vlanMin
)

func getTestVLAN(t *testing.T) (*net.IPNet, int) {
	vlanLock.Lock()
	defer vlanLock.Unlock()

	vlan := vlanNext
	vlanNext++

	subnet, err := cidr.Subnet(network, int(math.Ceil(math.Log2(vlanMax))), vlan)
	if err != nil {
		t.Error(err)
	}

	return subnet, vlan
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
func waitForUniFiAPI(ctx context.Context, client *unifi.Client, user, password string) error {
	maxRetries := 60
	retryDelay := 3 * time.Second

	log.Printf(
		"Waiting for UniFi API to be ready (max %d attempts, %v between attempts)...",
		maxRetries,
		retryDelay,
	)

	var loginSuccessful bool
	for i := range maxRetries {
		// Step 1: Try to login
		err := client.Login(ctx, user, password)
		if err != nil {
			errMsg := err.Error()
			if i < maxRetries-1 {
				if (i+1)%10 == 0 {
					log.Printf(
						"Still waiting for login... (attempt %d/%d): %v",
						i+1,
						maxRetries,
						errMsg,
					)
				}
				time.Sleep(retryDelay)
				continue
			}

			return fmt.Errorf(
				"UniFi API login did not succeed after %d attempts (waited %v): %w",
				maxRetries,
				time.Duration(maxRetries)*retryDelay,
				err,
			)
		}

		if !loginSuccessful {
			log.Printf("✓ Login successful after %d attempts", i+1)
			loginSuccessful = true
		}

		// Step 2: Verify sites are initialized
		sites, err := client.ListSites(ctx)
		if err != nil {

			if _, ok := err.(*unifi.LoginRequiredError); ok {
				if err = client.Login(ctx, user, password); err != nil {
					log.Println("Failed to login")
				}
				continue
			}

			// Not found errors are expected during initialization
			errStr := err.Error()
			if contains(errStr, "not found") || contains(errStr, "NotFound") ||
				contains(errStr, "404") {
				if i < maxRetries-1 {
					if (i+1)%10 == 0 {
						log.Printf(
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
						log.Printf(
							"Sites endpoint error... (attempt %d/%d): %v",
							i+1,
							maxRetries,
							err,
						)
					}
					time.Sleep(retryDelay)
					continue
				}

				return fmt.Errorf(
					"UniFi API sites not ready after %d attempts: %w",
					maxRetries,
					err,
				)
			}
		}

		if len(sites) == 0 {
			if i < maxRetries-1 {
				if (i+1)%10 == 0 {
					log.Printf(
						"No sites found yet... (attempt %d/%d)",
						i+1,
						maxRetries,
					)
				}
				time.Sleep(retryDelay)
				continue
			}

			return fmt.Errorf("no sites available after %d attempts", maxRetries)
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
						log.Printf(
							"Devices endpoint not ready... (attempt %d/%d): %v",
							i+1,
							maxRetries,
							err,
						)
					}
					time.Sleep(retryDelay)
					continue
				}

				return fmt.Errorf(
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
						log.Printf("Failed to adopt device %s: %v, retrying...", dev.MAC, adoptErr)
						// Retry adoption once after a brief delay
						time.Sleep(2 * time.Second)
						adoptErr = client.AdoptDevice(ctx, "default", dev.MAC)
						if adoptErr != nil {
							log.Printf("Failed to adopt device %s after retry: %v", dev.MAC, adoptErr)
						} else {
							log.Printf("Successfully adopted device %s on retry", dev.MAC)
						}
					} else {
						log.Printf("Successfully adopted device %s", dev.MAC)
					}
				}
			}
		}

		log.Printf(
			"✓ UniFi API fully ready (login + %d sites + devices endpoint) after %d attempts",
			len(sites),
			i+1,
		)
		return nil
	}

	return fmt.Errorf("UniFi API did not become ready after %d attempts", maxRetries)
}
