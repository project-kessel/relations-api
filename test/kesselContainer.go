package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/project-kessel/relations-api/internal/data"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	SpicedbSchemaBootstrapFile = "../internal/data/spicedb-test-data/basic_schema.zed"
	keycloakNetworkAlias       = "keycloak"
)

// LocalKesselContainer holds the testcontainers for the Kessel API, SpiceDB, and Keycloak
type LocalKesselContainer struct {
	Name               string
	logger             log.Logger
	HTTPport           string
	gRPCport           string
	KeycloakHTTPPort   string // host-mapped port for Keycloak HTTP (8080), for use in GetJWTToken etc.
	container          *testcontainers.DockerContainer
	spicedbContainer    *data.LocalSpiceDbContainer
	net                *testcontainers.DockerNetwork
	kccontainer        *testcontainers.DockerContainer
}

// CreateKesselAPIContainer creates the network, SpiceDB, Keycloak, and Kessel API containers using testcontainers-go
func CreateKesselAPIContainer(ctx context.Context, logger log.Logger) (_ *LocalKesselContainer, retErr error) {
	nw, err := network.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create network: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = nw.Remove(ctx)
		}
	}()

	spicedbContainer, err := data.CreateContainer(ctx, &data.ContainerOptions{Logger: logger, Network: nw})
	if err != nil {
		return nil, fmt.Errorf("could not create SpiceDB container: %w", err)
	}
	defer func() {
		if retErr != nil {
			spicedbContainer.Close()
		}
	}()

	kcCtr, err := testcontainers.Run(ctx, "quay.io/keycloak/keycloak:latest",
		testcontainers.WithCmd("start-dev", "--health-enabled=true"),
		testcontainers.WithExposedPorts("8080/tcp", "9000/tcp"),
		testcontainers.WithEnv(map[string]string{
			"KEYCLOAK_ADMIN":         "admin",
			"KEYCLOAK_ADMIN_PASSWORD": "admin",
		}),
		network.WithNetwork([]string{keycloakNetworkAlias}, nw),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/health").WithPort("9000/tcp").WithStartupTimeout(3*time.Minute),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not start Keycloak: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = testcontainers.TerminateContainer(kcCtr)
		}
	}()

	kcHTTPPort, err := kcCtr.MappedPort(ctx, "8080")
	if err != nil {
		return nil, fmt.Errorf("could not get Keycloak HTTP port: %w", err)
	}

	// JWKS URL for the Kessel API container (reaches Keycloak by network alias)
	kcurl := fmt.Sprintf("http://%s:8080/realms/master/protocol/openid-connect/certs", keycloakNetworkAlias)

	var (
		_, b, _, _ = runtime.Caller(0)
		basepath   = filepath.Dir(b)
	)
	schemaPath := filepath.Join(basepath, SpicedbSchemaBootstrapFile)
	schemaFile, err := os.Open(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("could not open schema file: %w", err)
	}
	defer func() { _ = schemaFile.Close() }()

	targetArch := runtime.GOARCH
	fromDockerfile := testcontainers.FromDockerfile{
		Context:    filepath.Join(basepath, ".."),
		Dockerfile: "Dockerfile",
		Repo:       "relations-api",
		Tag:        "test",
		BuildArgs:  map[string]*string{"TARGETARCH": &targetArch},
	}

	genericReq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: fromDockerfile,
		},
		Started: true,
	}
	for _, opt := range []testcontainers.ContainerCustomizer{
		testcontainers.WithEnv(map[string]string{
			"SPICEDB_ENDPOINT":    fmt.Sprintf("%s:50051", spicedbContainer.NetworkAlias()),
			"SPICEDB_PRESHARED":   "spicedbpreshared",
			"SPICEDB_JWKSURL":     kcurl,
			"SPICEDB_ENABLEAUTH":  "true",
			"SPICEDB_SCHEMA_FILE": "/tmp/spicedb_schema.zed",
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			Reader:            schemaFile,
			ContainerFilePath: "/tmp/spicedb_schema.zed",
			FileMode:          0o644,
		}),
		network.WithNetwork([]string{"rel"}, nw),
		testcontainers.WithExposedPorts("8000/tcp", "9000/tcp"),
	} {
		if err := opt.Customize(&genericReq); err != nil {
			return nil, fmt.Errorf("customize request: %w", err)
		}
	}

	ctr, err := testcontainers.GenericContainer(ctx, genericReq)
	if err != nil {
		if ctr != nil {
			_ = testcontainers.TerminateContainer(ctr)
		}
		return nil, fmt.Errorf("could not start Kessel API container: %w", err)
	}
	appCtr := ctr.(*testcontainers.DockerContainer)
	defer func() {
		if retErr != nil {
			_ = testcontainers.TerminateContainer(appCtr)
		}
	}()

	gRPCport, err := appCtr.MappedPort(ctx, "9000")
	if err != nil {
		return nil, fmt.Errorf("could not get gRPC port: %w", err)
	}
	httpPort, err := appCtr.MappedPort(ctx, "8000")
	if err != nil {
		return nil, fmt.Errorf("could not get HTTP port: %w", err)
	}

	return &LocalKesselContainer{
		Name:             appCtr.GetContainerID(),
		logger:           logger,
		container:        appCtr,
		gRPCport:         gRPCport.Port(),
		HTTPport:         httpPort.Port(),
		KeycloakHTTPPort: kcHTTPPort.Port(),
		spicedbContainer: spicedbContainer,
		net:              nw,
		kccontainer:      kcCtr,
	}, nil
}

// GetJWTToken obtains a JWT from Keycloak (baseURL should be the host-accessible URL, e.g. http://localhost:PORT)
func GetJWTToken(baseURL, username, password string) (*TokenResponse, error) {
	client := &http.Client{}
	form := url.Values{}
	form.Set("client_id", "admin-cli")
	form.Set("username", username)
	form.Set("password", password)
	form.Set("grant_type", "password")

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", baseURL), bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Errorf("error closing response: %v", err)
		}
	}()

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

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func (l *LocalKesselContainer) Close() {
	ctx := context.Background()
	if err := testcontainers.TerminateContainer(l.container); err != nil {
		log.NewHelper(l.logger).Error("Could not terminate Kessel API container. Please remove manually.")
	}
	l.spicedbContainer.Close()
	if err := testcontainers.TerminateContainer(l.kccontainer); err != nil {
		log.NewHelper(l.logger).Errorf("error terminating Keycloak container: %v", err)
	}
	if err := l.net.Remove(ctx); err != nil {
		log.NewHelper(l.logger).Errorf("could not remove network: %v", err)
	}
}
