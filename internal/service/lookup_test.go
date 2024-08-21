package service

import (
	"context"
	"os"
	"testing"

	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestLookupService_LookupSubjects_NoResults(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: simple_type("user"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: simple_type("thing"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	results := responseCollector.GetResponses()

	assert.Empty(t, results)
}

func TestLookupService_LookupResources_NoResults(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("workspace"), Id: "default"}},
		Relation: "view_the_thing",
		ResourceType: &v1beta1.ObjectType{
			Name:      "workspace",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	results := responseCollector.GetResponses()

	assert.Empty(t, results)
}

func TestLookupService_LookupSubjects_OneResult(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: simple_type("user"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: simple_type("thing"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"u1"}, ids)
}

func TestLookupService_LookupResources_OneResult(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("workspace"), Id: "default"}},
		Relation: "workspace",
		ResourceType: &v1beta1.ObjectType{
			Name:      "thing",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"thing1"}, ids)
}

func TestLookupService_LookupResources_TwoResults(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)
	//&v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("role_binding"), Id: "default_viewers"}}
	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject: &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("user"), Id: "u1"}},
		//Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("workspace"), Id: "default"}},
		Relation: "subject",
		ResourceType: &v1beta1.ObjectType{
			Name:      "role_binding",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"default_viewers", "default_viewers_two"}, ids)
}

func TestLookupService_LookupSubjects_TwoResults(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	err = seedThingInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u2")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: simple_type("user"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: simple_type("thing"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"u1", "u2"}, ids)
}

func createLookupService(spicedb *data.SpiceDbRepository) *LookupService {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	return NewLookupService(logger, biz.NewGetSubjectsUseCase(spicedb), biz.NewGetResourcesUseCase(spicedb))
}
func seedThingInDefaultWorkspace(ctx context.Context, spicedb *data.SpiceDbRepository, thing string) error {
	return spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("thing"), Id: thing},
			Relation: "workspace",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("workspace"), Id: "default"}},
		},
	}, biz.TouchSemantics(true))
}

func seedUserWithViewThingInDefaultWorkspace(ctx context.Context, spicedb *data.SpiceDbRepository, user string) error {
	return spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("role"), Id: "viewers"},
			Relation: "view_the_thing",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("user"), Id: "*"}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("role_binding"), Id: "default_viewers"},
			Relation: "subject",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("user"), Id: user}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("role_binding"), Id: "default_viewers_two"},
			Relation: "subject",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("user"), Id: user}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("role_binding"), Id: "default_viewers"},
			Relation: "granted",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("role"), Id: "viewers"}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: simple_type("workspace"), Id: "default"},
			Relation: "user_grant",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: simple_type("role_binding"), Id: "default_viewers"}},
		},
	}, biz.TouchSemantics(true))
}

func NewLookup_SubjectsServerStub(ctx context.Context) *Lookup_SubjectsServerStub {
	return &Lookup_SubjectsServerStub{
		ServerStream: nil,
		responses:    []*v1beta1.LookupSubjectsResponse{},
		ctx:          ctx,
	}
}

func NewLookup_ResourcesServerStub(ctx context.Context) *Lookup_ResourcesServerStub {
	return &Lookup_ResourcesServerStub{
		ServerStream: nil,
		responses:    []*v1beta1.LookupResourcesResponse{},
		ctx:          ctx,
	}
}

func (s *Lookup_SubjectsServerStub) GetResponses() []*v1beta1.LookupSubjectsResponse {
	return s.responses
}

func (s *Lookup_SubjectsServerStub) GetIDs() []string {
	ids := make([]string, len(s.responses))

	for i, r := range s.responses {
		ids[i] = r.Subject.Subject.Id
	}

	return ids
}

func (s *Lookup_ResourcesServerStub) GetIDs() []string {
	ids := make([]string, len(s.responses))

	for i, r := range s.responses {
		ids[i] = r.Resource.GetId()
	}

	return ids
}

type Lookup_SubjectsServerStub struct {
	grpc.ServerStream
	responses []*v1beta1.LookupSubjectsResponse
	ctx       context.Context
}

type Lookup_ResourcesServerStub struct {
	grpc.ServerStream
	responses []*v1beta1.LookupResourcesResponse
	ctx       context.Context
}

func (s *Lookup_SubjectsServerStub) Context() context.Context {
	return s.ctx
}

func (s *Lookup_SubjectsServerStub) Send(r *v1beta1.LookupSubjectsResponse) error {
	s.responses = append(s.responses, r)
	return nil
}

func (s *Lookup_ResourcesServerStub) GetResponses() []*v1beta1.LookupResourcesResponse {
	return s.responses
}

func (s *Lookup_ResourcesServerStub) Context() context.Context {
	return s.ctx
}

func (s *Lookup_ResourcesServerStub) Send(r *v1beta1.LookupResourcesResponse) error {
	s.responses = append(s.responses, r)
	return nil
}
