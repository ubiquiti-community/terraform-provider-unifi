package unifi

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/apparentlymart/go-cidr/cidr"
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

	dc, err := compose.NewDockerCompose("../docker-compose.yaml")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = dc.WithOsEnv().Up(ctx, compose.Wait(true)); err != nil {
		panic(err)
	}

	defer func() {
		if err := dc.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveImagesLocal); err != nil {
			panic(err)
		}
	}()

	container, err := dc.ServiceContainer(ctx, "unifi")
	if err != nil {
		panic(err)
	}

	lc := &logConsumer{StdOut: os.Getenv("UNIFI_STDOUT") == ""}

	testcontainers.WithLogConsumers(lc)

	endpoint, err := container.PortEndpoint(ctx, "8443/tcp", "https")
	if err != nil {
		panic(err)
	}

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

	testClient = &unifi.Client{}
	httpClient := &http.Client{}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient.Transport = transport
	testClient.SetHTTPClient(httpClient)
	testClient.SetBaseURL(endpoint)
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

// waitForUniFiAPI waits for the UniFi API to be ready and accepting JSON requests
// This is necessary because the container may report as healthy before the API is fully initialized.
func waitForUniFiAPI(ctx context.Context, client *unifi.Client, user, password string) error {
	maxRetries := 120
	retryDelay := 3 * time.Second

	testcontainers.Logger.Printf(
		"Waiting for UniFi API to be ready (max %d attempts, %v between attempts)...",
		maxRetries,
		retryDelay,
	)

	for i := range maxRetries {
		err := client.Login(ctx, user, password)
		if err == nil {
			testcontainers.Logger.Printf("✓ UniFi API is ready after %d attempts", i+1)
			return nil
		}

		// If we get a specific error indicating HTML response (setup wizard), keep waiting
		errMsg := err.Error()
		if i < maxRetries-1 {
			if (i+1)%10 == 0 {
				testcontainers.Logger.Printf(
					"Still waiting... (attempt %d/%d): %v",
					i+1,
					maxRetries,
					errMsg,
				)
			}
			time.Sleep(retryDelay)
			continue
		}

		return fmt.Errorf(
			"UniFi API did not become ready after %d attempts (waited %v): %w",
			maxRetries,
			time.Duration(maxRetries)*retryDelay,
			err,
		)
	}

	return fmt.Errorf("UniFi API did not become ready after %d attempts", maxRetries)
}
