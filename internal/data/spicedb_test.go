package data

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/stretchr/testify/mock"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"

	apiV1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var container *LocalSpiceDbContainer

func TestMain(m *testing.M) {
	var err error
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	container, err = CreateContainer(&ContainerOptions{Logger: logger})

	if err != nil {
		fmt.Printf("Error initializing Docker container: %s", err)
		os.Exit(-1)
	}

	result := m.Run()

	container.Close()
	os.Exit(result)
}

func TestCreateRelationship(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	exists := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.True(t, exists)
}

func TestCreateRelationshipWithConsistencyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	resp, err := spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	exists := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club",
		&apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: resp.GetConsistencyToken(),
			},
		},
	)
	assert.True(t, exists)
}

func TestCreateRelationshipWithSubjectRelation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role_binding", "fan_binding", "granted", "rbac", "role", "fan", ""),
		createRelationship("rbac", "role_binding", "fan_binding", "subject", "rbac", "group", "bob_club", "member"),
		createRelationship("rbac", "role", "fan", "view_widget", "rbac", "principal", "*", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	exists := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.True(t, exists)

	exists = CheckForRelationship(spiceDbRepo, "bob_club", "rbac", "group", "member", "subject", "rbac", "role_binding", "fan_binding", nil)
	assert.True(t, exists)

	// zed permission check rbac/role_binding:fan_binding subject rbac/principal:bob
	// bob is a subject of fan_binding
	runSpiceDBCheck(t, ctx, spiceDbRepo, "principal", "rbac", "bob", "subject", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_TRUE)

	// zed permission check rbac/role_binding:fan_binding subject rbac/principal:alice
	// alice is NOT a subject of fan_binding
	runSpiceDBCheck(t, ctx, spiceDbRepo, "principal", "rbac", "alice", "subject", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_FALSE)

	// zed permission check rbac/role_binding:fan_binding view_widget rbac/principal:bob
	// bob has view_widget permission
	runSpiceDBCheck(t, ctx, spiceDbRepo, "principal", "rbac", "bob", "view_widget", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_TRUE)

	// zed permission check rbac/role_binding:fan_binding subject rbac/principal:alice
	// alice does NOT have view_widget permission
	runSpiceDBCheck(t, ctx, spiceDbRepo, "principal", "rbac", "alice", "view_widget", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_FALSE)

	// zed permission check rbac/role_binding:fan_binding t_granted rbac/role:fan
	// check that role binding is tied to correct role
	runSpiceDBCheck(t, ctx, spiceDbRepo, "role", "rbac", "fan", "granted", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_TRUE)

	// zed permission check rbac/role_binding:fan_binding t_granted rbac/role:fake_fan
	// check for non-existent role not tied to role binding
	runSpiceDBCheck(t, ctx, spiceDbRepo, "role", "rbac", "fake_fan", "granted", "role_binding", "rbac", "fan_binding", apiV1beta1.CheckResponse_ALLOWED_FALSE)
}

func TestSecondCreateRelationshipFailsWithTouchFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Convert(err).Code())

	container.WaitForQuantizationInterval()

	exists := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.True(t, exists)
}

func TestSecondCreateRelationshipSucceedsWithTouchTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	touch = true

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	exists := CheckForRelationship(spiceDbRepo, "bob", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.True(t, exists)
}

type MockgRPCClientStream struct {
	mock.Mock
}

func (m *MockgRPCClientStream) SetHeader(md metadata.MD) error {
	panic("implement me")
}

func (m *MockgRPCClientStream) SendHeader(md metadata.MD) error {
	panic("implement me")
}

func (m *MockgRPCClientStream) SetTrailer(md metadata.MD) {
	panic("implement me")
}

func (m *MockgRPCClientStream) Context() context.Context {
	panic("implement me")
}

func (m *MockgRPCClientStream) SendMsg(_ any) error {
	panic("implement me")
}

func (m *MockgRPCClientStream) RecvMsg(_ any) error {
	panic("implement me")
}

func (m *MockgRPCClientStream) Recv() (*apiV1beta1.ImportBulkTuplesRequest, error) {
	args := m.Called()
	if req, ok := args.Get(0).(*apiV1beta1.ImportBulkTuplesRequest); ok {
		return req, args.Error(1)
	}
	return nil, args.Error(1)
}

// SendAndClose simulates sending a response and closing the stream
func (m *MockgRPCClientStream) SendAndClose(resp *apiV1beta1.ImportBulkTuplesResponse) error {
	args := m.Called(resp)
	return args.Error(0)
}

func (m *MockgRPCClientStream) CloseAndRecv() (*apiV1beta1.ImportBulkTuplesResponse, error) {
	args := m.Called()
	if res, ok := args.Get(0).(*apiV1beta1.ImportBulkTuplesResponse); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func TestImportBulkTuples(t *testing.T) {
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob5", ""),
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob3", ""),
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob6", ""),
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob9", ""),
	}

	mockgRPCClientStream := new(MockgRPCClientStream)
	mockgRPCClientStream.On("Recv").Return(&apiV1beta1.ImportBulkTuplesRequest{Tuples: rels}, nil).Once()
	mockgRPCClientStream.On("Recv").Return(nil, io.EOF).Once()
	mockgRPCClientStream.On("SendAndClose", &apiV1beta1.ImportBulkTuplesResponse{NumImported: uint64(len(rels))}).Return(nil)

	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = spiceDbRepo.ImportBulkTuples(mockgRPCClientStream)
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	exists := CheckForRelationship(spiceDbRepo, "bob5", "rbac", "principal", "", "member", "rbac", "group", "bob_club", nil)
	assert.True(t, exists)
}

func TestIsBackendAvailable(t *testing.T) {
	t.Parallel()

	spiceDbrepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = spiceDbrepo.IsBackendAvailable()
	assert.NoError(t, err)
}

func TestIsBackendUnavailable(t *testing.T) {
	t.Parallel()

	spiceDBRepo, _, err := NewSpiceDbRepository(&conf.Data{
		SpiceDb: &conf.Data_SpiceDb{
			Endpoint: "-1",
			Token:    "foobar",
			UseTLS:   true,
		}}, log.GetLogger())
	assert.NoError(t, err)

	err = spiceDBRepo.IsBackendAvailable()
	assert.Error(t, err)
}

func TestDoesNotCreateRelationshipWithSlashInSubjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badSubjectType := "special/user"

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", badSubjectType, "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.Error(t, err)
}

func TestDoesNotCreateRelationshipWithSlashInObjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badResourceType := "my/group"

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", badResourceType, "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.Error(t, err)
}

func TestCreateRelationshipFailsWithBadSubjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badSubjectType := "not_a_user"

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", badSubjectType, "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", "rbac", badSubjectType))
}

func TestCreateRelationshipFailsWithBadObjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badObjectType := "not_an_object"

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", badObjectType, "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	touch := biz.TouchSemantics(false)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", "rbac", badObjectType))

}

func TestSupportedNsTypeTupleFilterCombinationsInReadRelationships(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("bob_club"),
		ResourceType: pointerize("group"),
		Relation:     pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	assert.Error(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	assert.Error(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("principal"),
		},
	}, 0, "", nil)

	assert.Error(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
		},
	}, 0, "", nil)

	assert.Error(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	assert.NoError(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId: pointerize("bob_club"),
		Relation:   pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	assert.NoError(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId: pointerize("bob"),
		},
	}, 0, "", nil)

	assert.NoError(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId: pointerize("bob_club"),
		Relation:   pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId: pointerize("bob"),
		},
	}, 0, "", nil)

	assert.NoError(t, err)

	_, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId: pointerize("bob"),
		},
	}, 0, "", nil)

	assert.NoError(t, err)
}

func TestWriteAndReadBackRelationships(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	readRelChan, _, err := spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	if !assert.NoError(t, err) {
		return
	}

	readrels := spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 1, len(readrels))
}

func TestWriteReadBackDeleteAndReadBackRelationships(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	readRelChan, _, err := spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	if !assert.NoError(t, err) {
		return
	}

	readrels := spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 1, len(readrels))

	_, err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, nil)

	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	readRelChan, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", nil)

	if !assert.NoError(t, err) {
		return
	}

	readrels = spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 0, len(readrels))

}

func TestWriteReadBackDeleteAndReadBackRelationships_WithConsistencyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
	}

	respCreate, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	readRelChan, _, err := spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", &apiV1beta1.Consistency{
		Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
			AtLeastAsFresh: respCreate.GetConsistencyToken(),
		},
	})

	if !assert.NoError(t, err) {
		return
	}

	readrels := spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 1, len(readrels))

	respDelete, err := spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, nil)

	if !assert.NoError(t, err) {
		return
	}

	readRelChan, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, 0, "", &apiV1beta1.Consistency{
		Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
			AtLeastAsFresh: respDelete.GetConsistencyToken(),
		},
	})

	if !assert.NoError(t, err) {
		return
	}

	readrels = spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 0, len(readrels))

}

func TestSpiceDbRepository_CheckPermission_WithConsistencyToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "principal", Namespace: "rbac",
			},
			Id: "bob",
		},
	}

	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Name: "workspace", Namespace: "rbac",
		},
		Id: "test",
	}
	// no wait, immediately read after write.
	// zed permission check rbac/workspace:test view_widget rbac/principal:bob --explain
	check := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_widget",
		Resource: resource,
		Consistency: &apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(), // pass createRelationship consistency token
			},
		},
	}
	resp, err := spiceDbRepo.Check(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse := apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned consistency token may not be same as created consistency token.
	}
	assert.Equal(t, &checkResponse, resp)

}

func TestSpiceDbRepository_CheckForUpdatePermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "principal", Namespace: "rbac",
			},
			Id: "bob",
		},
	}

	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Name: "workspace", Namespace: "rbac",
		},
		Id: "test",
	}
	// no wait, immediately read after write.
	// zed permission check rbac/workspace:test view_widget rbac/principal:bob --explain
	check := apiV1beta1.CheckForUpdateRequest{
		Subject:  subject,
		Relation: "view_widget",
		Resource: resource,
	}
	resp, err := spiceDbRepo.CheckForUpdate(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckForUpdateResponse_ALLOWED_TRUE
	checkResponse := apiV1beta1.CheckForUpdateResponse{
		Allowed:          apiV1beta1.CheckForUpdateResponse_ALLOWED_TRUE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned ConsistencyToken may not be same as created ConsistencyToken.
	}
	assert.Equal(t, &checkResponse, resp)

}

func TestSpiceDbRepository_CheckPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "principal", Namespace: "rbac",
			},
			Id: "bob",
		},
	}

	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Name: "workspace", Namespace: "rbac",
		},
		Id: "test",
	}
	// zed permission check rbac/workspace:test view_widget rbac/principal:bob --explain
	check := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_widget",
		Resource: resource,
	}
	resp, err := spiceDbRepo.Check(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	dummyConsistencyToken := "AAAAAAAAHHHHH"
	checkResponse := apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken},
	}
	resp.ConsistencyToken = &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken}
	assert.Equal(t, &checkResponse, resp)

	//Remove // rbac/role_binding:rb_test#t_subject@rbac/principal:bob
	_, err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("rb_test"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("role_binding"),
		Relation:          pointerize("subject"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, nil)
	if !assert.NoError(t, err) {
		return
	}

	// zed permission check rbac/workspace:test view_widget rbac/principal:bob --explain
	check2 := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_widget",
		Resource: resource,
	}

	resp2, err := spiceDbRepo.Check(ctx, &check2)
	if !assert.NoError(t, err) {
		return
	}
	dummyConsistencyToken = "AAAAAAAAHHHHH"
	checkResponsev2 := apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_FALSE,
		ConsistencyToken: &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken},
	}
	resp2.ConsistencyToken = &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken}
	assert.Equal(t, &checkResponsev2, resp2)
}

func TestSpiceDbRepository_NewEnemyProblem_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "u1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "u2", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	// u1
	u1Check := apiV1beta1.CheckRequest{
		Subject: &apiV1beta1.SubjectReference{
			Subject: &apiV1beta1.ObjectReference{
				Type: &apiV1beta1.ObjectType{
					Name: "principal", Namespace: "rbac",
				},
				Id: "u1",
			},
		},
		Relation: "view_widget",
		Resource: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "workspace", Namespace: "rbac",
			},
			Id: "test",
		},
		Consistency: &apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(), // pass createRelationship consistency token
			},
		},
	}

	// no wait, immediately read after write.
	// zed permission check rbac/workspace:test user_grant rbac/principal:u1 --explain
	resp, err := spiceDbRepo.Check(ctx, &u1Check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse := apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned ConsistencyToken may not be same as created ConsistencyToken.
	}
	assert.Equal(t, &checkResponse, resp)

	// u2
	u2Check := apiV1beta1.CheckRequest{
		Subject: &apiV1beta1.SubjectReference{
			Subject: &apiV1beta1.ObjectReference{
				Type: &apiV1beta1.ObjectType{
					Name: "principal", Namespace: "rbac",
				},
				Id: "u2",
			},
		},
		Relation: "view_widget",
		Resource: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "workspace", Namespace: "rbac",
			},
			Id: "test",
		},
		Consistency: &apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(), // pass createRelationship consistency token
			},
		},
	}

	// zed permission check rbac/workspace:test user_grant rbac/principal:u2 --explain
	resp, err = spiceDbRepo.Check(ctx, &u2Check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse = apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned ConsistencyToken may not be same as created ConsistencyToken.
	}
	assert.Equal(t, &checkResponse, resp)

	// remove access from u1, keep access for u2.
	respDelete, err := spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("rb_test"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("role_binding"),
		Relation:          pointerize("subject"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("u1"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, nil)
	if !assert.NoError(t, err) {
		return
	}

	// ensure u1 no longer has access, while u2 still does.

	// zed permission check rbac/workspace:test user_grant rbac/principal:u1 --explain
	u1Check.Consistency = &apiV1beta1.Consistency{
		Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
			AtLeastAsFresh: respDelete.GetConsistencyToken(), // pass createRelationship consistency token
		},
	}
	resp, err = spiceDbRepo.Check(ctx, &u1Check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_FALSE
	checkResponse = apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_FALSE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned ConsistencyToken may not be same as created ConsistencyToken.
	}
	assert.Equal(t, &checkResponse, resp)

	// zed permission check rbac/workspace:test user_grant rbac/principal:u2 --explain
	u2Check.Consistency = &apiV1beta1.Consistency{
		Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
			AtLeastAsFresh: respDelete.GetConsistencyToken(), // pass deleteRelationship consistency token
		},
	}
	resp, err = spiceDbRepo.Check(ctx, &u2Check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse = apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: resp.GetConsistencyToken(), // returned ConsistencyToken may not be same as created ConsistencyToken.
	}
	assert.Equal(t, &checkResponse, resp)
}

// Test is ambiguous as consistency token may not be *strictly* used.
// if a better revision is available and faster than it will be used, causing
// race conditions for this test to fail
// func TestSpiceDbRepository_NewEnemyProblem_Failure(t *testing.T) {
// 	t.Parallel()

// 	ctx := context.Background()
// 	spiceDbRepo, err := container.CreateSpiceDbRepository()
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	rels := []*apiV1beta1.Relationship{
// 		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
// 		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
// 		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
// 		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "u1", ""),
// 		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "u2", ""),
// 		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
// 	}

// 	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	// u1
// 	u1Check := apiV1beta1.CheckRequest{
// 		Subject: &apiV1beta1.SubjectReference{
// 			Subject: &apiV1beta1.ObjectReference{
// 				Type: &apiV1beta1.ObjectType{
// 					Name: "principal", Namespace: "rbac",
// 				},
// 				Id: "u1",
// 			},
// 		},
// 		Relation: "view_widget",
// 		Resource: &apiV1beta1.ObjectReference{
// 			Type: &apiV1beta1.ObjectType{
// 				Name: "workspace", Namespace: "rbac",
// 			},
// 			Id: "test",
// 		},
// 		Consistency: &apiV1beta1.Consistency{
// 			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
// 				AtLeastAsFresh: relationshipResp.GetConsistencyToken(), // pass createRelationship consistency token
// 			},
// 		},
// 	}

// 	// u2
// 	u2Check := apiV1beta1.CheckRequest{
// 		Subject: &apiV1beta1.SubjectReference{
// 			Subject: &apiV1beta1.ObjectReference{
// 				Type: &apiV1beta1.ObjectType{
// 					Name: "principal", Namespace: "rbac",
// 				},
// 				Id: "u2",
// 			},
// 		},
// 		Relation: "view_widget",
// 		Resource: &apiV1beta1.ObjectReference{
// 			Type: &apiV1beta1.ObjectType{
// 				Name: "workspace", Namespace: "rbac",
// 			},
// 			Id: "test",
// 		},
// 		Consistency: &apiV1beta1.Consistency{
// 			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
// 				AtLeastAsFresh: relationshipResp.GetConsistencyToken(), // pass createRelationship consistency token
// 			},
// 		},
// 	}

// 	// remove access from u1, keep access for u2.
// 	_, err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
// 		ResourceId:        pointerize("rb_test"),
// 		ResourceNamespace: pointerize("rbac"),
// 		ResourceType:      pointerize("role_binding"),
// 		Relation:          pointerize("subject"),
// 		SubjectFilter: &apiV1beta1.SubjectFilter{
// 			SubjectId:        pointerize("u1"),
// 			SubjectNamespace: pointerize("rbac"),
// 			SubjectType:      pointerize("principal"),
// 		},
// 	})
// 	if !assert.NoError(t, err) {
// 		return
// 	}

// 	// u1 has access even though we removed access. u2 still has access.

// 	// zed permission check rbac/workspace:test user_grant rbac/principal:u1 --explain
// 	resp, err := spiceDbRepo.Check(ctx, &u1Check) // we're passing a ConsistencyToken revision before deletion occurred.
// 	if !assert.NoError(t, err) {
// 		return
// 	}
// 	//apiV1.CheckResponse_ALLOWED_TRUE
// 	checkResponse := apiV1beta1.CheckResponse{
// 		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
// 		ConsistencyToken: resp.GetConsistencyToken(), // returned consistency token may not be same as created consistency token.
// 	}

// 	if spiceDbRepo.fullyConsistent { // new enemy problem doesn't apply if we're fully consistent.
// 		checkResponse.Allowed = apiV1beta1.CheckResponse_ALLOWED_FALSE
// 		assert.Equal(t, &checkResponse, resp)
// 	} else { // we technically dont have access, but according to consistency token revision we do!
// 		assert.Equal(t, &checkResponse, resp) // we expect true even with removed access.
// 	}

// 	// zed permission check rbac/workspace:test user_grant rbac/principal:u2 --explain
// 	resp, err = spiceDbRepo.Check(ctx, &u2Check)
// 	if !assert.NoError(t, err) {
// 		return
// 	}
// 	//apiV1.CheckResponse_ALLOWED_TRUE
// 	checkResponse = apiV1beta1.CheckResponse{
// 		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
// 		ConsistencyToken: resp.GetConsistencyToken(), // returned consistency token may not be same as created consistency token.
// 	}
// 	assert.Equal(t, &checkResponse, resp)
// }

func TestSpiceDbRepository_CheckPermission_MinimizeLatency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: "principal", Namespace: "rbac",
			},
			Id: "bob",
		},
	}

	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Name: "workspace", Namespace: "rbac",
		},
		Id: "test",
	}

	// Test with minimize_latency = True.

	// zed permission check rbac/workspace:test view_widget rbac/principal:bob --explain
	check := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_widget",
		Resource: resource,
		Consistency: &apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_MinimizeLatency{
				MinimizeLatency: true,
			},
		},
	}
	resp, err := spiceDbRepo.Check(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	dummyConsistencyToken := "AAAAAAAAHHHHH"
	checkResponse := apiV1beta1.CheckResponse{
		Allowed:          apiV1beta1.CheckResponse_ALLOWED_TRUE,
		ConsistencyToken: &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken},
	}
	resp.ConsistencyToken = &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken}
	assert.Equal(t, &checkResponse, resp)

	//Remove // rbac/role_binding:rb_test#t_subject@rbac/principal:bob
	_, err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("rb_test"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("role_binding"),
		Relation:          pointerize("subject"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}, nil)
	if !assert.NoError(t, err) {
		return
	}
}

func TestSpiceDbRepository_CheckBulk(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "bob_club", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "rb_test", ""),
		createRelationship("rbac", "role_binding", "rb_test", "granted", "rbac", "role", "rl1", ""),
		createRelationship("rbac", "role_binding", "rb_test", "subject", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "role", "rl1", "view_widget", "rbac", "principal", "*", ""),
	}

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true), nil)
	if !assert.NoError(t, err) {
		return
	}

	container.WaitForQuantizationInterval()

	items := []*apiV1beta1.CheckBulkRequestItem{
		{
			Resource: &apiV1beta1.ObjectReference{
				Type: &apiV1beta1.ObjectType{
					Name:      "workspace",
					Namespace: "rbac",
				},
				Id: "test",
			},
			Relation: "view_widget",
			Subject: &apiV1beta1.SubjectReference{
				Subject: &apiV1beta1.ObjectReference{
					Type: &apiV1beta1.ObjectType{
						Name:      "principal",
						Namespace: "rbac",
					},
					Id: "bob",
				},
			},
		},
		{
			Resource: &apiV1beta1.ObjectReference{
				Type: &apiV1beta1.ObjectType{
					Name:      "workspace",
					Namespace: "rbac",
				},
				Id: "test",
			},
			Relation: "view_widget",
			Subject: &apiV1beta1.SubjectReference{
				Subject: &apiV1beta1.ObjectReference{
					Type: &apiV1beta1.ObjectType{
						Name:      "principal",
						Namespace: "rbac",
					},
					Id: "alice",
				},
			},
		},
	}

	req := &apiV1beta1.CheckBulkRequest{
		Items: items,
	}
	resp, err := spiceDbRepo.CheckBulk(ctx, req)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Equal(t, len(items), len(resp.GetPairs())) {
		return
	}

	results := map[string]apiV1beta1.CheckBulkResponseItem_Allowed{}
	for _, p := range resp.GetPairs() {
		subjId := p.GetRequest().GetSubject().GetSubject().GetId()
		results[subjId] = p.GetItem().GetAllowed()
	}
	assert.Equal(t, apiV1beta1.CheckBulkResponseItem_ALLOWED_TRUE, results["bob"])
	assert.Equal(t, apiV1beta1.CheckBulkResponseItem_ALLOWED_FALSE, results["alice"])
}

func TestFromSpicePair_WithError(t *testing.T) {
	t.Parallel()

	// Build a SpiceDB pair that contains an error instead of an item
	pair := &v1.CheckBulkPermissionsPair{
		Request: &v1.CheckBulkPermissionsRequestItem{
			Resource: &v1.ObjectReference{
				ObjectType: "rbac/workspace",
				ObjectId:   "test",
			},
			Permission: "view_widget",
			Subject: &v1.SubjectReference{
				Object: &v1.ObjectReference{
					ObjectType: "rbac/principal",
					ObjectId:   "bob",
				},
			},
		},
		Response: &v1.CheckBulkPermissionsPair_Error{
			Error: &rpcstatus.Status{
				Code:    int32(codes.InvalidArgument),
				Message: "invalid request",
			},
		},
	}

	got := fromSpicePair(pair, log.NewHelper(log.DefaultLogger))
	assert.NotNil(t, got)
	// When error is present, the oneof response should be set to error and item should be nil
	assert.Nil(t, got.GetItem())
	if assert.NotNil(t, got.GetError()) {
		assert.Equal(t, int32(codes.InvalidArgument), got.GetError().GetCode())
		assert.Equal(t, "invalid request", got.GetError().GetMessage())
	}

	// And the request should be preserved/mapped back correctly
	req := got.GetRequest()
	assert.Equal(t, "rbac", req.GetResource().GetType().GetNamespace())
	assert.Equal(t, "workspace", req.GetResource().GetType().GetName())
	assert.Equal(t, "test", req.GetResource().GetId())
	assert.Equal(t, "view_widget", req.GetRelation())
	assert.Equal(t, "rbac", req.GetSubject().GetSubject().GetType().GetNamespace())
	assert.Equal(t, "principal", req.GetSubject().GetSubject().GetType().GetName())
	assert.Equal(t, "bob", req.GetSubject().GetSubject().GetId())
}

func TestSpiceDbRepository_CreateRelationships_WithFencing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	// Acquire a lock to get a fencing token
	lockIdentifier := "test-lock-1"
	lockResp, err := spiceDbRepo.AcquireLock(ctx, lockIdentifier)
	assert.NoError(t, err)
	fencingToken := lockResp.GetLockToken()
	assert.NotEmpty(t, fencingToken)

	// Create a relationship with fencing
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "fencing_group", "member", "rbac", "principal", "fenced_bob", ""),
	}
	touch := biz.TouchSemantics(false)
	fencing := &apiV1beta1.FencingCheck{
		LockId:    lockIdentifier,
		LockToken: fencingToken,
	}
	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, fencing)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	// Relationship should exist
	exists := CheckForRelationship(
		spiceDbRepo, "fenced_bob", "rbac", "principal", "", "member", "rbac", "group", "fencing_group", nil,
	)
	assert.True(t, exists)

	// Try to create with an invalid fencing token
	badFencing := &apiV1beta1.FencingCheck{
		LockId:    lockIdentifier,
		LockToken: "invalid-token",
	}
	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, badFencing)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error writing relationships to SpiceDB")

	// try to create with a non-existent lock id
	badFencing2 := &apiV1beta1.FencingCheck{
		LockId:    "invalid-lock-id",
		LockToken: fencingToken,
	}
	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, badFencing2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error writing relationships to SpiceDB")
}

func TestSpiceDbRepository_DeleteRelationships_WithFencing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	// Acquire a lock to get a fencing token
	lockIdentifier := "test-lock-1"
	lockResp, err := spiceDbRepo.AcquireLock(ctx, lockIdentifier)
	assert.NoError(t, err)
	fencingToken := lockResp.GetLockToken()
	assert.NotEmpty(t, fencingToken)

	// Create a relationship to delete
	rels := []*apiV1beta1.Relationship{
		createRelationship("rbac", "group", "fencing_group_del", "member", "rbac", "principal", "fenced_bob_del", ""),
	}
	touch := biz.TouchSemantics(false)
	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch, nil)
	assert.NoError(t, err)

	// Delete with correct fencing
	filter := &apiV1beta1.RelationTupleFilter{
		ResourceId:        pointerize("fencing_group_del"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:        pointerize("fenced_bob_del"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}
	fencing := &apiV1beta1.FencingCheck{
		LockId:    lockIdentifier,
		LockToken: fencingToken,
	}
	_, err = spiceDbRepo.DeleteRelationships(ctx, filter, fencing)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	// Relationship should not exist
	exists := CheckForRelationship(
		spiceDbRepo, "fenced_bob_del", "rbac", "principal", "", "member", "rbac", "group", "fencing_group_del", nil,
	)
	assert.False(t, exists)

	// Try to delete with an invalid fencing token
	_, err = spiceDbRepo.DeleteRelationships(ctx, filter, &apiV1beta1.FencingCheck{
		LockId:    lockIdentifier,
		LockToken: "invalid-token",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error invoking DeleteRelationships in SpiceDB")

	// try to delete with a non-existent lock id
	_, err = spiceDbRepo.DeleteRelationships(ctx, filter, &apiV1beta1.FencingCheck{
		LockId:    "invalid-lock-id",
		LockToken: fencingToken,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error invoking DeleteRelationships in SpiceDB")
}

func TestSpiceDbRepository_AcquireLock_NewLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	identifier := "test-lock-1"

	// Acquire a new lock
	resp, err := spiceDbRepo.AcquireLock(ctx, identifier)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.GetLockToken())

}

func TestSpiceDbRepository_AcquireLock_ReplaceExistingLock(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	identifier := "test-lock-1"

	// Acquire initial lock
	resp1, err := spiceDbRepo.AcquireLock(ctx, identifier)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp1.GetLockToken())

	// Acquire lock again, forcefully replacing the existing lock
	resp2, err := spiceDbRepo.AcquireLock(ctx, identifier)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2.GetLockToken())
	assert.NotEqual(t, resp1.GetLockToken(), resp2.GetLockToken())
}

func TestSpiceDbRepository_AcquireLock_EmptyIdentifier(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	// Try to acquire lock with an empty identifier
	_, err = spiceDbRepo.AcquireLock(ctx, "")
	assert.Error(t, err)
}

func TestSpiceDbRepository_LookupResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	// Create a permission structure with multiple widgets:
	// - alice has view access to widgets in workspace1
	// - charlie has view access to widgets in workspace2
	// - widget1, widget2, widget3 are in workspace1
	// - widget4 is in workspace2
	rels := []*apiV1beta1.Relationship{
		// Workspace grants
		createRelationship("rbac", "workspace", "workspace1", "user_grant", "rbac", "role_binding", "binding1", ""),
		createRelationship("rbac", "workspace", "workspace2", "user_grant", "rbac", "role_binding", "binding2", ""),

		// Role binding to role
		createRelationship("rbac", "role_binding", "binding1", "granted", "rbac", "role", "viewer", ""),
		createRelationship("rbac", "role_binding", "binding2", "granted", "rbac", "role", "viewer", ""),

		// Role binding to subjects
		createRelationship("rbac", "role_binding", "binding1", "subject", "rbac", "principal", "alice", ""),
		createRelationship("rbac", "role_binding", "binding2", "subject", "rbac", "principal", "charlie", ""),

		// Role permissions
		createRelationship("rbac", "role", "viewer", "view_widget", "rbac", "principal", "*", ""),

		// Widgets in workspaces
		createRelationship("rbac", "widget", "widget1", "workspace", "rbac", "workspace", "workspace1", ""),
		createRelationship("rbac", "widget", "widget2", "workspace", "rbac", "workspace", "workspace1", ""),
		createRelationship("rbac", "widget", "widget3", "workspace", "rbac", "workspace", "workspace1", ""),
		createRelationship("rbac", "widget", "widget4", "workspace", "rbac", "workspace", "workspace2", ""),
	}

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, true, nil)
	if !assert.NoError(t, err) {
		return
	}

	// Test 1: LookupResources to find all widgets that alice can view
	// alice should see widget1, widget2, widget3 (from workspace1)
	resourceType := &apiV1beta1.ObjectType{
		Namespace: "rbac",
		Name:      "widget",
	}
	aliceSubject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "principal",
			},
			Id: "alice",
		},
	}

	resources, errs, err := spiceDbRepo.LookupResources(
		ctx,
		resourceType,
		"view", // relation/permission
		aliceSubject,
		0,  // limit (0 = no limit)
		"", // continuation
		&apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(),
			},
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	// Collect all resources from the channel
	foundResources := make(map[string]bool)
	for {
		select {
		case res, ok := <-resources:
			if !ok {
				goto checkResults
			}
			foundResources[res.Resource.Id] = true
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatalf("Error receiving resources: %v", err)
			}
		}
	}

checkResults:
	// alice should see widget1, widget2, widget3 from workspace1
	assert.True(t, foundResources["widget1"], "alice should have view permission on widget1")
	assert.True(t, foundResources["widget2"], "alice should have view permission on widget2")
	assert.True(t, foundResources["widget3"], "alice should have view permission on widget3")
	assert.False(t, foundResources["widget4"], "alice should not have view permission on widget4")
	assert.Equal(t, 3, len(foundResources), "alice should find exactly 3 widgets with view permission")

	// Test 2: LookupResources to find all widgets that charlie can view
	// charlie should only see widget4 (from workspace2)
	charlieSubject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "principal",
			},
			Id: "charlie",
		},
	}

	resources2, errs2, err := spiceDbRepo.LookupResources(
		ctx,
		resourceType,
		"view",
		charlieSubject,
		10, // specify a limit to test that it gets passed through (unlike LookupSubjects)
		"",
		&apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(),
			},
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	foundResources2 := make(map[string]bool)
	for {
		select {
		case res, ok := <-resources2:
			if !ok {
				goto checkResults2
			}
			foundResources2[res.Resource.Id] = true
		case err, ok := <-errs2:
			if ok && err != nil {
				t.Fatalf("Error receiving resources: %v", err)
			}
		}
	}

checkResults2:
	// charlie should only see widget4 from workspace2
	assert.False(t, foundResources2["widget1"], "charlie should not have view permission on widget1")
	assert.False(t, foundResources2["widget2"], "charlie should not have view permission on widget2")
	assert.False(t, foundResources2["widget3"], "charlie should not have view permission on widget3")
	assert.True(t, foundResources2["widget4"], "charlie should have view permission on widget4")
	assert.Equal(t, 1, len(foundResources2), "charlie should find exactly 1 widget with view permission")
}

func TestSpiceDbRepository_LookupSubjects(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	// Create a permission structure:
	// - alice and bob are members of the group "admins"
	// - charlie is a member of the group "viewers"
	// - workspace "test" has user_grant role_binding "admin_binding" and "viewer_binding"
	// - "admin_binding" grants role "admin" to group "admins" members
	// - "viewer_binding" grants role "viewer" to group "viewers" members
	// - role "admin" has view_widget and use_widget permissions
	// - role "viewer" has view_widget permission only
	rels := []*apiV1beta1.Relationship{
		// Group memberships
		createRelationship("rbac", "group", "admins", "member", "rbac", "principal", "alice", ""),
		createRelationship("rbac", "group", "admins", "member", "rbac", "principal", "bob", ""),
		createRelationship("rbac", "group", "viewers", "member", "rbac", "principal", "charlie", ""),

		// Workspace grants
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "admin_binding", ""),
		createRelationship("rbac", "workspace", "test", "user_grant", "rbac", "role_binding", "viewer_binding", ""),

		// Role binding to role
		createRelationship("rbac", "role_binding", "admin_binding", "granted", "rbac", "role", "admin", ""),
		createRelationship("rbac", "role_binding", "viewer_binding", "granted", "rbac", "role", "viewer", ""),

		// Role binding to subjects (groups)
		createRelationship("rbac", "role_binding", "admin_binding", "subject", "rbac", "group", "admins", "member"),
		createRelationship("rbac", "role_binding", "viewer_binding", "subject", "rbac", "group", "viewers", "member"),

		// Role permissions
		createRelationship("rbac", "role", "admin", "view_widget", "rbac", "principal", "*", ""),
		createRelationship("rbac", "role", "admin", "use_widget", "rbac", "principal", "*", ""),
		createRelationship("rbac", "role", "viewer", "view_widget", "rbac", "principal", "*", ""),
	}

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, true, nil)
	if !assert.NoError(t, err) {
		return
	}

	// LookupSubjects to find all principals that have view_widget permission on workspace:test
	subjectType := &apiV1beta1.ObjectType{
		Namespace: "rbac",
		Name:      "principal",
	}
	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "workspace",
		},
		Id: "test",
	}

	subjects, errs, err := spiceDbRepo.LookupSubjects(
		ctx,
		subjectType,
		"",            // subject_relation (empty for direct principals)
		"view_widget", // relation/permission
		resource,
		0,  // limit (0 = no limit)
		"", // continuation
		&apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(),
			},
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	// Collect all subjects from the channel
	foundSubjects := make(map[string]bool)
	for {
		select {
		case subj, ok := <-subjects:
			if !ok {
				// Channel closed, we're done
				goto checkResults
			}
			foundSubjects[subj.Subject.Subject.Id] = true
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatalf("Error receiving subjects: %v", err)
			}
		}
	}

checkResults:
	// Verify that alice, bob, and charlie all have view_widget permission
	assert.True(t, foundSubjects["alice"], "alice should have view_widget permission")
	assert.True(t, foundSubjects["bob"], "bob should have view_widget permission")
	assert.True(t, foundSubjects["charlie"], "charlie should have view_widget permission")
	assert.Equal(t, 3, len(foundSubjects), "should find exactly 3 subjects with view_widget permission")

	// Now test LookupSubjects for use_widget permission (only alice and bob via admin role)
	subjects2, errs2, err := spiceDbRepo.LookupSubjects(
		ctx,
		subjectType,
		"",
		"use_widget",
		resource,
		0,
		"",
		&apiV1beta1.Consistency{
			Requirement: &apiV1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: relationshipResp.GetConsistencyToken(),
			},
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	foundSubjects2 := make(map[string]bool)
	for {
		select {
		case subj, ok := <-subjects2:
			if !ok {
				goto checkResults2
			}
			foundSubjects2[subj.Subject.Subject.Id] = true
		case err, ok := <-errs2:
			if ok && err != nil {
				t.Fatalf("Error receiving subjects: %v", err)
			}
		}
	}

checkResults2:
	// Verify that only alice and bob have use_widget permission (not charlie)
	assert.True(t, foundSubjects2["alice"], "alice should have use_widget permission")
	assert.True(t, foundSubjects2["bob"], "bob should have use_widget permission")
	assert.False(t, foundSubjects2["charlie"], "charlie should not have use_widget permission")
	assert.Equal(t, 2, len(foundSubjects2), "should find exactly 2 subjects with use_widget permission")
}

func pointerize(value string) *string { //Used to turn string literals into pointers
	return &value
}

func runSpiceDBCheck(t *testing.T, ctx context.Context, spiceDbRepo *SpiceDbRepository, subjectType,
	subjectNamespace, subjectID, relation, resourceType, resourceNamespace, resourceID string,
	expectedAllowed apiV1beta1.CheckResponse_Allowed) {
	check := apiV1beta1.CheckRequest{
		Subject: &apiV1beta1.SubjectReference{
			Subject: &apiV1beta1.ObjectReference{
				Type: &apiV1beta1.ObjectType{
					Name:      subjectType,
					Namespace: subjectNamespace,
				},
				Id: subjectID,
			},
		},
		Relation: relation,
		Resource: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name:      resourceType,
				Namespace: resourceNamespace,
			},
			Id: resourceID,
		},
	}

	resp, err := spiceDbRepo.Check(ctx, &check)
	assert.NoError(t, err)

	dummyConsistencyToken := "AAAAAAAAHHHHH"
	expectedResponse := apiV1beta1.CheckResponse{
		Allowed:          expectedAllowed,
		ConsistencyToken: &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken},
	}
	resp.ConsistencyToken = &apiV1beta1.ConsistencyToken{Token: dummyConsistencyToken}
	assert.Equal(t, &expectedResponse, resp)
}

func createRelationship(resourceNamespace string, resourceType string, resourceId string, relationship string, subjectNamespace string, subjectType string, subjectId string, subjectRelationship string) *apiV1beta1.Relationship {
	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: &apiV1beta1.ObjectType{
				Name: subjectType, Namespace: subjectNamespace,
			},
			Id: subjectId,
		},
	}

	if subjectRelationship != "" {
		subject.Relation = &subjectRelationship
	}

	resource := &apiV1beta1.ObjectReference{
		Type: &apiV1beta1.ObjectType{
			Name: resourceType, Namespace: resourceNamespace,
		},
		Id: resourceId,
	}

	return &apiV1beta1.Relationship{
		Resource: resource,
		Relation: relationship,
		Subject:  subject,
	}
}

func spiceRelChanToSlice(c chan *biz.RelationshipResult) []*biz.RelationshipResult {
	s := make([]*biz.RelationshipResult, 0)
	for i := range c {
		s = append(s, i)
	}
	return s
}
