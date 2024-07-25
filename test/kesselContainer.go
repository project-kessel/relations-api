package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/project-kessel/relations-api/internal/data"
)

// LocalKesselContainer struct that holds pointers to the localKesselContainer, dockertest pool and exposes the port
type LocalKesselContainer struct {
	Name             string
	logger           log.Logger
	HTTPport         string
	gRPCport         string
	container        *dockertest.Resource
	pool             *dockertest.Pool
	spicedbContainer *data.LocalSpiceDbContainer
	network          string
	kccontainer      *dockertest.Resource
}

func CreateKesselAPIContainer(logger log.Logger) (*LocalKesselContainer, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %spicedb", err)
		return nil, err
	}

	// Create a custom network
	networkName := "kessel-network"
	findNetwork := func(name string) (*docker.Network, error) {
		networks, err := pool.Client.ListNetworks()
		if err != nil {
			return nil, err
		}
		for _, net := range networks {
			if net.Name == name {
				return &net, nil
			}
		}
		return nil, nil
	}

	// Check if network exists
	network, err := findNetwork(networkName)
	if err != nil {
		log.Infof("Could not list networks: %s", err)
	}
	if network == nil {
		network, err = pool.Client.CreateNetwork(docker.CreateNetworkOptions{
			Name:           networkName,
			Driver:         "bridge",
			CheckDuplicate: true,
		})
		if err != nil {
			log.Fatalf("Could not create network: %s", err)
		}
	}

	pool.MaxWait = 3 * time.Minute
	// Create a custom network
	container, err := data.CreateContainer(&data.ContainerOptions{Logger: logger, Network: network})
	if err != nil {
		fmt.Printf("Error initializing Docker localKesselContainer: %s", err)
		os.Exit(-1)
	}
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	options := &dockertest.RunOptions{
		Repository: "quay.io/keycloak/keycloak",
		Tag:        "latest",
		Cmd: []string{
			"start-dev",
			"--health-enabled=true",
		},
		ExposedPorts: []string{"8080/tcp", "9000/tcp"},
		Env: []string{
			"KEYCLOAK_ADMIN=admin",
			"KEYCLOAK_ADMIN_PASSWORD=admin",
		},
		NetworkID: network.ID,
	}

	kcresource, err := pool.RunWithOptions(options, func(config *docker.HostConfig) {
		config.NetworkMode = "bridge"
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start keycloak resource: %s", err)
	}

	portMetric := kcresource.GetPort("9000/tcp")
	kcname := strings.Trim(kcresource.Container.Name, "/")
	kcurl := fmt.Sprintf("http://%s:%s/realms/master/protocol/openid-connect/certs", kcname, "8080")

	// Wait for the container to be ready
	if err := pool.Retry(func() error {
		err = checkKeycloakHealth(fmt.Sprintf("http://localhost:%s", portMetric))
		log.Info("Waiting for Keycloak to be ready...")
		time.Sleep(20 * time.Second)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to keycloak container: %s", err)
	}

	targetArch := "amd64" // or "arm64", depending on your needs
	enable_auth := fmt.Sprintf("SPICEDB_ENABLEAUTH=%t", true)
	sso := fmt.Sprintf("SPICEDB_JWKSURL=%s", kcurl)
	name := strings.Trim(container.Name(), "/")
	endpoint := fmt.Sprintf("SPICEDB_ENDPOINT=%s:%s", name, "50051")
	presharedSecret := fmt.Sprintf("SPICEDB_PRESHARED=%s", "spicedbpreshared")

	resource, err := pool.BuildAndRunWithBuildOptions(&dockertest.BuildOptions{
		Dockerfile: "Dockerfile", // Path to your Dockerfile
		ContextDir: "../",        // Context directory for the Dockerfile
		Platform:   "linux/amd64",
		BuildArgs: []docker.BuildArg{
			{Name: "TARGETARCH", Value: targetArch},
		},
	}, &dockertest.RunOptions{
		Name:      "rel",
		Env:       []string{endpoint, presharedSecret, sso, enable_auth},
		NetworkID: network.ID,
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err.Error())
	}

	gRPCport := resource.GetPort("9000/tcp")
	httpPort := resource.GetPort("8000/tcp")

	return &LocalKesselContainer{
		Name:             resource.Container.Name,
		kccontainer:      kcresource,
		container:        resource,
		gRPCport:         gRPCport,
		HTTPport:         httpPort,
		spicedbContainer: container,
		logger:           logger,
		pool:             pool,
		network:          network.ID,
	}, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func checkKeycloakHealth(baseURL string) error {
	resp, err := http.Get(fmt.Sprintf("%s/health", baseURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
func GetJWTToken(baseURL, username, password string) (*TokenResponse, error) {
	client := &http.Client{}
	data := url.Values{}
	data.Set("client_id", "admin-cli")
	data.Set("username", username)
	data.Set("password", password)
	data.Set("grant_type", "password")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", baseURL), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func (l *LocalKesselContainer) Close() {
	err := l.pool.Purge(l.container)
	if err != nil {
		log.NewHelper(l.logger).Error("Could not purge Kessel Container from test. Please delete manually.")
	}
	l.spicedbContainer.Close()
	l.kccontainer.Close()
	if err := l.pool.Client.RemoveNetwork(l.network); err != nil {
		log.Fatalf("Could not remove network: %s", err)
	}
}
