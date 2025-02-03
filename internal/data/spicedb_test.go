package data

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	resp, err := spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	touch = true

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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
	})

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

	respCreate, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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
	})

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

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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

	_, err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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
	})
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

	relationshipResp, err := spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
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
	})
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

// Test is amibguous as consistency token may not be *strictly* used.
// if a better revision is available and faster than it will be used, causing
// race conditions for this test to failure
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
