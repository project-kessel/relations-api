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
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	// SpicedbImage is the image used for containerized spiceDB in tests
	SpicedbImage = "authzed/spicedb"
	// SpicedbVersion is the image version used for containerized spiceDB in tests
	SpicedbVersion = "v1.37.0"
	// SpicedbSchemaBootstrapFile specifies an optional bootstrap schema file to be used for testing
	SpicedbSchemaBootstrapFile = "spicedb-test-data/basic_schema.zed"
	// SpicedbRelationsBootstrapFile specifies an optional bootstrap file containing relations to be used for testing
	SpicedbRelationsBootstrapFile = ""
	// FullyConsistent specifices the consistency mode used for our read API calls
	// may experience different results between tests and manual probing if the values differ
	FullyConsistent = false // Should probably be inline with our config file. (TODO: Can we make our tests grab the same value?)
)

// LocalSpiceDbContainer struct that holds pointers to the container, dockertest pool and exposes the port
type LocalSpiceDbContainer struct {
	logger         log.Logger
	port           string
	container      *dockertest.Resource
	pool           *dockertest.Pool
	name           string
	schemaLocation string
}

type ContainerOptions struct {
	Logger  log.Logger
	Network *docker.Network
}

// CreateContainer creates a new SpiceDbContainer using dockertest
func CreateContainer(opts *ContainerOptions) (*LocalSpiceDbContainer, error) {
	pool, err := dockertest.NewPool("") // Empty string uses default docker env
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	pool.MaxWait = 3 * time.Minute

	var (
		_, b, _, _ = runtime.Caller(0)
		basepath   = filepath.Dir(b)
	)

	cmd := []string{"serve-testing", "--skip-release-check=true"}

	runopt := &dockertest.RunOptions{
		Repository:   SpicedbImage,
		Tag:          SpicedbVersion, // Replace this with an actual version
		Cmd:          cmd,
		ExposedPorts: []string{"50051/tcp", "50052/tcp"},
	}
	if opts.Network != nil {
		runopt.NetworkID = opts.Network.ID
	}
	resource, err := pool.RunWithOptions(runopt)

	if err != nil {
		return nil, fmt.Errorf("could not start spicedb resource: %w", err)
	}

	port := resource.GetPort("50051/tcp")

	// Give the service time to boot.
	cErr := pool.Retry(func() error {
		log.NewHelper(opts.Logger).Info("Attempting to connect to spicedb...")

		conn, err := grpc.NewClient(
			fmt.Sprintf("localhost:%s", port),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("error connecting to spiceDB: %v", err.Error())
		}

		client := grpc_health_v1.NewHealthClient(conn)
		_, err = client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
		return err
	})

	if cErr != nil {
		return nil, cErr
	}

	return &LocalSpiceDbContainer{
		name:           resource.Container.Name,
		logger:         opts.Logger,
		port:           port,
		container:      resource,
		pool:           pool,
		schemaLocation: path.Join(basepath, SpicedbSchemaBootstrapFile),
	}, nil
}

// Port returns the Port the container is listening
func (l *LocalSpiceDbContainer) Port() string {
	return l.port
}

// Name returns the container name
func (l *LocalSpiceDbContainer) Name() string {
	return l.name
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

// CreateClient creates a new client that connects to the dockerized spicedb instance and the right store
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

	defer os.RemoveAll(tmpDir)

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

// Close purges the container
func (l *LocalSpiceDbContainer) Close() {
	err := l.pool.Purge(l.container)
	if err != nil {
		log.NewHelper(l.logger).Error("Could not purge SpiceDB Container from test. Please delete manually.")
	}
}

// CheckForRelationship returns true if the given subject has the given relationship to the given resource, otherwise false
func CheckForRelationship(client biz.ZanzibarRepository, subjectID string, subjectNamespace string, subjectType string, subjectRelationship string, relationship string, resourceNamespace string, resourceType string, resourceID string) bool {
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
	}, 1, biz.ContinuationToken(""))

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
