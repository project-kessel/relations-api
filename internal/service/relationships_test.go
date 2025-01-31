package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var container *data.LocalSpiceDbContainer

func TestMain(m *testing.M) {
	var err error
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	container, err = data.CreateContainer(&data.ContainerOptions{Logger: logger})

	if err != nil {
		fmt.Printf("Error initializing Docker container: %s", err)
		os.Exit(-1)
	}

	result := m.Run()

	container.Close()
	os.Exit(result)
}

func TestRelationshipsService_CreateRelationships(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)
	ctx := context.Background()
	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")

	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	reqCopy := proto.Clone(req)
	_, err = relationshipsService.CreateTuples(ctx, reqCopy.(*v1beta1.CreateTuplesRequest))
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	readReq := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
	}
	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(readReq, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responseRelationships := collectingServer.responses

	for _, resp := range responseRelationships {
		assert.Equal(t, expected.Resource.Id, resp.Tuple.Resource.Id)
		assert.Equal(t, expected.Resource.Type.Namespace, resp.Tuple.Resource.Type.Namespace)
		assert.Equal(t, expected.Resource.Type.Name, resp.Tuple.Resource.Type.Name)
		assert.Equal(t, expected.Subject.Subject.Id, resp.Tuple.Subject.Subject.Id)
		assert.Equal(t, expected.Subject.Subject.Type.Namespace, resp.Tuple.Subject.Subject.Type.Namespace)
		assert.Equal(t, expected.Subject.Subject.Type.Name, resp.Tuple.Subject.Subject.Type.Name)
		assert.Equal(t, expected.Relation, resp.Tuple.Relation)
	}

}

func TestRelationshipsService_CreateRelationships_WithZookie(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)
	ctx := context.Background()
	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")

	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	reqCopy := proto.Clone(req)
	resp, err := relationshipsService.CreateTuples(ctx, reqCopy.(*v1beta1.CreateTuplesRequest))
	assert.NoError(t, err)

	readReq := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
		Zookie: resp.GetCreatedAt(),
	}
	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(readReq, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responseRelationships := collectingServer.responses

	for _, resp := range responseRelationships {
		assert.Equal(t, expected.Resource.Id, resp.Tuple.Resource.Id)
		assert.Equal(t, expected.Resource.Type.Namespace, resp.Tuple.Resource.Type.Namespace)
		assert.Equal(t, expected.Resource.Type.Name, resp.Tuple.Resource.Type.Name)
		assert.Equal(t, expected.Subject.Subject.Id, resp.Tuple.Subject.Subject.Id)
		assert.Equal(t, expected.Subject.Subject.Type.Namespace, resp.Tuple.Subject.Subject.Type.Namespace)
		assert.Equal(t, expected.Subject.Subject.Type.Name, resp.Tuple.Subject.Subject.Type.Name)
		assert.Equal(t, expected.Relation, resp.Tuple.Relation)
	}
}

func TestRelationshipsService_CreateRelationshipsWithTouchFalse(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)

	ctx := context.Background()
	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")
	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	reqCopy := proto.Clone(req)
	_, err = relationshipsService.CreateTuples(ctx, reqCopy.(*v1beta1.CreateTuplesRequest))
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	readReq := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
	}
	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(readReq, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responseRelationships := collectingServer.responses

	for _, resp := range responseRelationships {
		assert.Equal(t, expected.Resource.Id, resp.Tuple.Resource.Id)
		assert.Equal(t, expected.Resource.Type.Namespace, resp.Tuple.Resource.Type.Namespace)
		assert.Equal(t, expected.Resource.Type.Name, resp.Tuple.Resource.Type.Name)
		assert.Equal(t, expected.Subject.Subject.Id, resp.Tuple.Subject.Subject.Id)
		assert.Equal(t, expected.Subject.Subject.Type.Namespace, resp.Tuple.Subject.Subject.Type.Namespace)
		assert.Equal(t, expected.Subject.Subject.Type.Name, resp.Tuple.Subject.Subject.Type.Name)
		assert.Equal(t, expected.Relation, resp.Tuple.Relation)
	}

	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.Equal(t, codes.AlreadyExists, status.Convert(err).Code())

}

// nil tuples in CreateRelationshipsRequest should be equivalent to an empty list of tuples (and not error)
func TestRelationshipsService_CreateRelationshipsWithNilRelationshipsSlice(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)
	ctx := context.Background()

	req := &v1beta1.CreateTuplesRequest{
		Tuples: nil,
	}
	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.NoError(t, err)
}

func TestRelationshipsService_CreateRelationshipsWithBadSubjectType(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)
	ctx := context.Background()
	badSubjectType := rbac_ns_type("not_a_user")
	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", badSubjectType, "bob", "")
	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", badSubjectType.GetNamespace(), badSubjectType.GetName()))
}

func TestRelationshipsService_CreateRelationshipsWithBadObjectType(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)
	ctx := context.Background()
	badObjectType := rbac_ns_type("not_an_object")
	expected := createRelationship(badObjectType, "bob_club", "member", rbac_ns_type("user"), "bob", "")
	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Convert(err).Code())
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", badObjectType.GetNamespace(), badObjectType.GetName()))
}

func TestRelationshipsService_DeleteRelationships(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)

	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")

	ctx := context.Background()
	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.NoError(t, err)

	delreq := &v1beta1.DeleteTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}}
	_, err = relationshipsService.DeleteTuples(ctx, delreq)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	readReq := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
	}

	container.WaitForQuantizationInterval()

	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(readReq, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responses := collectingServer.responses

	assert.Equal(t, 0, len(responses))
	assert.NoError(t, err)
}

func TestRelationshipsService_DeleteRelationships_WithZookie(t *testing.T) {
	t.Parallel()
	err, relationshipsService := setup(t)
	assert.NoError(t, err)

	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")

	ctx := context.Background()
	req := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, req)
	assert.NoError(t, err)

	delreq := &v1beta1.DeleteTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	}}
	resp, err := relationshipsService.DeleteTuples(ctx, delreq)
	assert.NoError(t, err)

	readReq := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
		Zookie: resp.GetDeletedAt(),
	}

	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(readReq, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responses := collectingServer.responses

	assert.Equal(t, 0, len(responses))
	assert.NoError(t, err)
}

func setup(t *testing.T) (error, *RelationshipsService) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	spiceDbRepository, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	createRelationshipsUsecase := biz.NewCreateRelationshipsUsecase(spiceDbRepository, logger)
	readRelationshipsUsecase := biz.NewReadRelationshipsUsecase(spiceDbRepository, logger)
	deleteRelationshipsUsecase := biz.NewDeleteRelationshipsUsecase(spiceDbRepository, logger)
	importBulkUsecase := biz.NewImportBulkTuplesUsecase(spiceDbRepository, logger)
	relationshipsService := NewRelationshipsService(logger, createRelationshipsUsecase, readRelationshipsUsecase, deleteRelationshipsUsecase, importBulkUsecase)
	return err, relationshipsService
}

func TestRelationshipsService_ReadRelationships(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	spiceDbRepository, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	createRelationshipsUsecase := biz.NewCreateRelationshipsUsecase(spiceDbRepository, logger)
	readRelationshipsUsecase := biz.NewReadRelationshipsUsecase(spiceDbRepository, logger)
	deleteRelationshipsUsecase := biz.NewDeleteRelationshipsUsecase(spiceDbRepository, logger)
	bulkImportTuplesUsecase := biz.NewImportBulkTuplesUsecase(spiceDbRepository, logger)
	relationshipsService := NewRelationshipsService(logger, createRelationshipsUsecase, readRelationshipsUsecase, deleteRelationshipsUsecase, bulkImportTuplesUsecase)

	expected := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")

	reqCr := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, reqCr)
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	req := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceId:        pointerize("bob_club"),
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
	}

	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	err = relationshipsService.ReadTuples(req, collectingServer)
	if err != nil {
		t.FailNow()
	}
	responses := collectingServer.responses

	assert.Equal(t, 1, len(responses))
	assert.NoError(t, err)
}

func TestRelationshipsService_ReadRelationships_Paginated(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	spiceDbRepository, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	createRelationshipsUsecase := biz.NewCreateRelationshipsUsecase(spiceDbRepository, logger)
	readRelationshipsUsecase := biz.NewReadRelationshipsUsecase(spiceDbRepository, logger)
	deleteRelationshipsUsecase := biz.NewDeleteRelationshipsUsecase(spiceDbRepository, logger)
	bulkImportTuplesUsecase := biz.NewImportBulkTuplesUsecase(spiceDbRepository, logger)
	relationshipsService := NewRelationshipsService(logger, createRelationshipsUsecase, readRelationshipsUsecase, deleteRelationshipsUsecase, bulkImportTuplesUsecase)

	expected1 := createRelationship(rbac_ns_type("group"), "bob_club", "member", rbac_ns_type("principal"), "bob", "")
	expected2 := createRelationship(rbac_ns_type("group"), "other_bob_club", "member", rbac_ns_type("principal"), "bob", "")

	reqCr := &v1beta1.CreateTuplesRequest{
		Tuples: []*v1beta1.Relationship{
			expected1,
			expected2,
		},
	}
	_, err = relationshipsService.CreateTuples(ctx, reqCr)
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	req := &v1beta1.ReadTuplesRequest{Filter: &v1beta1.RelationTupleFilter{
		ResourceNamespace: pointerize("rbac"),
		ResourceType:      pointerize("group"),
		Relation:          pointerize("member"),
		SubjectFilter: &v1beta1.SubjectFilter{
			SubjectId:        pointerize("bob"),
			SubjectNamespace: pointerize("rbac"),
			SubjectType:      pointerize("principal"),
		},
	},
		Pagination: &v1beta1.RequestPagination{
			Limit: 1,
		},
	}

	collectingServer := NewRelationships_ReadRelationshipsServerStub(ctx)
	for {
		beforeLength := len(collectingServer.responses)
		err = relationshipsService.ReadTuples(req, collectingServer)
		if err != nil {
			t.FailNow()
		}
		afterLength := len(collectingServer.responses)

		assert.GreaterOrEqual(t, 1, afterLength-beforeLength)

		if beforeLength == afterLength {
			break
		}

		req.Pagination.ContinuationToken = collectingServer.GetLatestContinuation()
	}

	assert.Equal(t, 2, len(collectingServer.responses))
	assert.NoError(t, err)
}

func rbac_ns_type(typename string) *v1beta1.ObjectType {
	return &v1beta1.ObjectType{Name: typename, Namespace: "rbac"}
}

func pointerize(value string) *string { //Used to turn string literals into pointers
	return &value
}

func createRelationship(objectType *v1beta1.ObjectType, objectId string, relationship string, subjectType *v1beta1.ObjectType, subjectId string, subjectRelationship string) *v1beta1.Relationship {
	subject := &v1beta1.SubjectReference{
		Subject: &v1beta1.ObjectReference{
			Type: subjectType,
			Id:   subjectId,
		},
	}

	if subjectRelationship != "" {
		subject.Relation = &subjectRelationship
	}

	resource := &v1beta1.ObjectReference{
		Type: objectType,
		Id:   objectId,
	}

	return &v1beta1.Relationship{
		Resource: resource,
		Relation: relationship,
		Subject:  subject,
	}
}

// Below is the boilerplate for creating test servers for streaming ReadTuples rpc

func NewRelationships_ReadRelationshipsServerStub(ctx context.Context) *Relationships_ReadRelationshipsServerStub {
	return &Relationships_ReadRelationshipsServerStub{
		ServerStream: nil,
		responses:    []*v1beta1.ReadTuplesResponse{},
		ctx:          ctx,
	}
}

type Relationships_ReadRelationshipsServerStub struct {
	grpc.ServerStream
	responses []*v1beta1.ReadTuplesResponse
	ctx       context.Context
}

func (x *Relationships_ReadRelationshipsServerStub) GetDistinctTuples() []*v1beta1.Relationship {
	set := make(map[*v1beta1.Relationship]bool)

	for _, response := range x.responses {
		set[response.Tuple] = true
	}

	results := make([]*v1beta1.Relationship, 0, len(set))
	for tuple, found := range set {
		if !found {
			continue
		}

		results = append(results, tuple)
	}

	return results
}

func (x *Relationships_ReadRelationshipsServerStub) GetLatestContinuation() *string {
	if len(x.responses) == 0 {
		return nil
	}

	response := x.responses[len(x.responses)-1]
	return &response.Pagination.ContinuationToken
}

func (x *Relationships_ReadRelationshipsServerStub) Send(m *v1beta1.ReadTuplesResponse) error {
	x.responses = append(x.responses, m)
	return nil
}

func (x *Relationships_ReadRelationshipsServerStub) Context() context.Context {
	return x.ctx
}
