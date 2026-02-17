package data

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// SpicedbImage is the image used for containerized spiceDB in tests
	SpicedbImage = "authzed/spicedb"
	// SpicedbVersion is the image version used for containerized spiceDB in tests
	SpicedbVersion = "v1.47.1"
	// SpicedbSchemaBootstrapFile specifies an optional bootstrap schema file to be used for testing
	SpicedbSchemaBootstrapFile = "spicedb-test-data/basic_schema.zed"
	// SpicedbRelationsBootstrapFile specifies an optional bootstrap file containing relations to be used for testing
	SpicedbRelationsBootstrapFile = ""
	// FullyConsistent specifices the consistency mode used for our read API calls
	// may experience different results between tests and manual probing if the values differ
	FullyConsistent = false // Should probably be inline with our config file. (TODO: Can we make our tests grab the same value?)
)

const spicedbNetworkAlias = "spicedb"

// LocalSpiceDbContainer struct that holds the testcontainers container and exposes the port
type LocalSpiceDbContainer struct {
	logger         log.Logger
	port           string
	container      *testcontainers.DockerContainer
	name           string
	networkAlias   string
	schemaLocation string
}

// ContainerOptions configures SpiceDB container creation
type ContainerOptions struct {
	Logger  log.Logger
	Network *testcontainers.DockerNetwork
}

// CreateContainer creates a new SpiceDbContainer using testcontainers-go
func CreateContainer(ctx context.Context, opts *ContainerOptions) (*LocalSpiceDbContainer, error) {
	var (
		_, b, _, _ = runtime.Caller(0)
		basepath   = filepath.Dir(b)
	)

	image := SpicedbImage + ":" + SpicedbVersion
	runOpts := []testcontainers.ContainerCustomizer{
		testcontainers.WithCmd("serve-testing", "--skip-release-check=true"),
		testcontainers.WithExposedPorts("50051/tcp", "50052/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("50051/tcp").WithStartupTimeout(3 * time.Minute),
		),
	}

	if opts.Network != nil {
		runOpts = append(runOpts,
			network.WithNetwork([]string{spicedbNetworkAlias}, opts.Network),
		)
	}

	ctr, err := testcontainers.Run(ctx, image, runOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not start spicedb resource: %w", err)
	}

	port, err := ctr.MappedPort(ctx, "50051")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)
		return nil, fmt.Errorf("could not get spicedb port: %w", err)
	}

	inspect, err := ctr.Inspect(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)
		return nil, fmt.Errorf("could not inspect spicedb container: %w", err)
	}
	name := inspect.Name

	alias := ""
	if opts.Network != nil {
		alias = spicedbNetworkAlias
	}

	return &LocalSpiceDbContainer{
		name:           name,
		networkAlias:   alias,
		logger:         opts.Logger,
		port:           port.Port(),
		container:      ctr,
		schemaLocation: path.Join(basepath, SpicedbSchemaBootstrapFile),
	}, nil
}

// Port returns the port the container is listening on (host-mapped)
func (l *LocalSpiceDbContainer) Port() string {
	return l.port
}

// Name returns the actual container name assigned by Docker
func (l *LocalSpiceDbContainer) Name() string {
	return l.name
}

// NetworkAlias returns the network alias used for inter-container communication,
// or empty if no network is configured
func (l *LocalSpiceDbContainer) NetworkAlias() string {
	return l.networkAlias
}

// NewToken returns a new token used for the container so a new store is created in serve-testing
func (l *LocalSpiceDbContainer) NewToken() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// WaitForQuantizationInterval needed to avoid read-before-write when loading the schema
func (l *LocalSpiceDbContainer) WaitForQuantizationInterval() {
	if !FullyConsistent {
		time.Sleep(10 * time.Millisecond)
	}
}

// CreateSpiceDbRepository creates a repository that connects to the containerized SpiceDB instance
func (l *LocalSpiceDbContainer) CreateSpiceDbRepository() (*SpiceDbRepository, error) {
	randomKey, err := l.NewToken()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "relations-api")
	if err != nil {
		return nil, err
	}
	tmpFile, err := os.CreateTemp(tmpDir, "spicedbpreshared")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(tmpFile.Name(), []byte(randomKey), 0666)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.NewHelper(l.logger).Errorf("error removing temporary directory: %w", err)
		}
	}()

	spiceDbConf := &conf.Data_SpiceDb{
		UseTLS:          false,
		Endpoint:        "localhost:" + l.port,
		Token:           tmpFile.Name(),
		SchemaFile:      l.schemaLocation,
		FullyConsistent: FullyConsistent, // Should be inline with our config file
	}
	repo, _, err := NewSpiceDbRepository(&conf.Data{SpiceDb: spiceDbConf}, l.logger)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// Close terminates the container
func (l *LocalSpiceDbContainer) Close() {
	if err := testcontainers.TerminateContainer(l.container); err != nil {
		log.NewHelper(l.logger).Error("Could not terminate SpiceDB container. Please remove manually.")
	}
}

// CheckForRelationship returns true if the given subject has the given relationship to the given resource, otherwise false
func CheckForRelationship(client biz.ZanzibarRepository, subjectID string, subjectNamespace string, subjectType string, subjectRelationship string, relationship string, resourceNamespace string, resourceType string, resourceID string, consistency *v1beta1.Consistency) bool {
	ctx := context.TODO()

	var subjectRelationRef *string = nil //Relation is optional
	if subjectRelationship != "" {
		subjectRelationRef = &subjectRelationship
	}

	results, errors, err := client.ReadRelationships(ctx, &v1beta1.RelationTupleFilter{
		ResourceNamespace: &resourceNamespace,
		ResourceType:      &resourceType,
		ResourceId:        &resourceID,
		Relation:          &relationship,
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectNamespace: &subjectNamespace,
			SubjectType:      &subjectType,
			SubjectId:        &subjectID,
			Relation:         subjectRelationRef,
		},
	}, 1, biz.ContinuationToken(""), consistency)

	if err != nil {
		panic(err)
	}

	found := false
	select {
	case err, ok := <-errors:
		if ok {
			panic(err)
		}
	case _, ok := <-results:
		if ok {
			found = true
		}
	}

	return found
}
