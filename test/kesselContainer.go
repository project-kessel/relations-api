package test

import (
	"fmt"
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
	targetArch := "amd64" // or "arm64", depending on your needs

	enable_auth := fmt.Sprintf("SPICEDB_ENABLEAUTH=%t", true)
	sso := fmt.Sprintf("SPICEDB_JWKSURL=%s", "http://host.docker.internal:8180/jwks")
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
		container:        resource,
		gRPCport:         gRPCport,
		HTTPport:         httpPort,
		spicedbContainer: container,
		logger:           logger,
		pool:             pool,
		network:          network.ID,
	}, nil
}

func (l *LocalKesselContainer) Close() {
	err := l.pool.Purge(l.container)
	if err != nil {
		log.NewHelper(l.logger).Error("Could not purge Kessel Container from test. Please delete manually.")
	}
	l.spicedbContainer.Close()
	if err := l.pool.Client.RemoveNetwork(l.network); err != nil {
		log.Fatalf("Could not remove network: %s", err)
	}
}
