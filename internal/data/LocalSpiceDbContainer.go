package data

import (
	"ciam-rebac/internal/conf"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/authzed/authzed-go/v1"
	"github.com/go-kratos/kratos/v2/log"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ory/dockertest"
)

const (
	// SpicedbImage is the image used for containerized spiceDB in tests
	SpicedbImage = "authzed/spicedb"
	// SpicedbVersion is the image version used for containerized spiceDB in tests
	SpicedbVersion = "v1.22.2"
	// SpicedbSchemaBootstrapFile specifies an optional bootstrap schema file to be used for testing
	SpicedbSchemaBootstrapFile = "spicedb-test-data/basic_schema.yaml"
	// SpicedbRelationsBootstrapFile specifies an optional bootstrap file containing relations to be used for testing
	SpicedbRelationsBootstrapFile = ""
)

// LocalSpiceDbContainer struct that holds pointers to the container, dockertest pool and exposes the port
type LocalSpiceDbContainer struct {
	logger    log.Logger
	port      string
	container *dockertest.Resource
	pool      *dockertest.Pool
}

// CreateContainer creates a new SpiceDbContainer using dockertest
func CreateContainer(logger log.Logger) (*LocalSpiceDbContainer, error) {
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

	var mounts []string
	if SpicedbSchemaBootstrapFile != "" {
		cmd = append(cmd, "--load-configs")
		cmd = append(cmd, "/mnt/spicedb_bootstrap.yaml")
		mounts = append(mounts, path.Join(basepath, SpicedbSchemaBootstrapFile)+":/mnt/spicedb_bootstrap.yaml")
	}
	if SpicedbRelationsBootstrapFile != "" {
		if SpicedbSchemaBootstrapFile != "" {
			cmd[len(cmd)-1] = "/mnt/spicedb_bootstrap.yaml,/mnt/spicedb_bootstrap_relations.yaml"
		} else {
			cmd = append(cmd, "--load-configs")
			cmd = append(cmd, "/mnt/spicedb_bootstrap_relations.yaml")
		}
		mounts = append(mounts, path.Join(basepath, SpicedbRelationsBootstrapFile)+":/mnt/spicedb_bootstrap_relations.yaml")
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   SpicedbImage,
		Tag:          SpicedbVersion, // Replace this with an actual version
		Cmd:          cmd,
		Mounts:       mounts,
		ExposedPorts: []string{"50051/tcp", "50052/tcp"},
	})

	if err != nil {
		return nil, fmt.Errorf("could not start spicedb resource: %w", err)
	}

	port := resource.GetPort("50051/tcp")

	// Give the service time to boot.
	cErr := pool.Retry(func() error {
		log.NewHelper(logger).Info("Attempting to connect to spicedb...")

		conn, err := grpc.Dial(
			fmt.Sprintf("localhost:%s", port),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return fmt.Errorf("error connecting to spiceDB: %v", err.Error())
		}

		client := v1.NewSchemaServiceClient(conn)

		//read scheme we add via mount
		_, err = client.ReadSchema(context.Background(), &v1.ReadSchemaRequest{})

		return err
	})

	if cErr != nil {
		return nil, cErr
	}

	return &LocalSpiceDbContainer{
		logger:    logger,
		port:      port,
		container: resource,
		pool:      pool,
	}, nil
}

// Port returns the Port the container is listening
func (l *LocalSpiceDbContainer) Port() string {
	return l.port
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
	time.Sleep(10 * time.Millisecond)
}

// CreateClient creates a new client that connects to the dockerized spicedb instance and the right store
func (l *LocalSpiceDbContainer) CreateSpiceDbRepository() (*SpiceDbRepository, error) {
	randomKey, err := l.NewToken()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "rebac")
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
		UseTLS:   false,
		Endpoint: "localhost:" + l.port,
		Token:    tmpFile.Name(),
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
func CheckForRelationship(client *authzed.Client, subjectID string, subjectType string, subjectRelationship string, relationship string, resourceType string, resourceID string) bool {
	ctx := context.TODO()
	resp, err := client.ReadRelationships(ctx, &v1.ReadRelationshipsRequest{
		Consistency: &v1.Consistency{Requirement: &v1.Consistency_FullyConsistent{FullyConsistent: true}},
		RelationshipFilter: &v1.RelationshipFilter{
			ResourceType:       resourceType,
			OptionalResourceId: resourceID,
			OptionalRelation:   relationship,
			OptionalSubjectFilter: &v1.SubjectFilter{
				SubjectType:       subjectType,
				OptionalSubjectId: subjectID,
				OptionalRelation:  &v1.SubjectFilter_RelationFilter{Relation: subjectRelationship},
			},
		},
		OptionalLimit: 1,
	})

	if err != nil {
		panic(err)
	}

	_, e := resp.Recv()

	if errors.Is(e, io.EOF) {
		return false
	}
	// error
	if e != nil {
		panic(e)
	}
	return true
}
