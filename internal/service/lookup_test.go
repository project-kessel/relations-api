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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: rbac_ns_type("principal"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	results := responseCollector.GetResponses()

	assert.Empty(t, results)
}

func TestLookupService_LookupSubjects_NoResults_WithConsistencyToken(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	resp, err := seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: rbac_ns_type("principal"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
		Consistency: &v1beta1.Consistency{
			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: resp.GetConsistencyToken(),
			},
		},
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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
		Relation: "view_widget",
		ResourceType: &v1beta1.ObjectType{
			Name:      "workspace",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	results := responseCollector.GetResponses()

	assert.Empty(t, results)
}

func TestLookupService_LookupResources_NoResults_WithConsistencyToken(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	resp, err := seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
		Relation: "view_widget",
		ResourceType: &v1beta1.ObjectType{
			Name:      "workspace",
			Namespace: "rbac",
		},
		Consistency: &v1beta1.Consistency{
			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: resp.GetConsistencyToken(),
			},
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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	_, err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: rbac_ns_type("principal"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"u1"}, ids)
}

func TestLookupService_LookupSubjects_OneResult_WithConsistencyToken(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	resp, err := seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: rbac_ns_type("principal"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
		Consistency: &v1beta1.Consistency{
			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: resp.GetConsistencyToken(),
			},
		},
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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
		Relation: "workspace",
		ResourceType: &v1beta1.ObjectType{
			Name:      "widget",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"thing1"}, ids)
}
func TestLookupService_LookupResources_OneResult_WithConsistencyToken(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	resp, err := seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)

	service := createLookupService(spicedb)

	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
		Relation: "workspace",
		ResourceType: &v1beta1.ObjectType{
			Name:      "widget",
			Namespace: "rbac",
		},
		Consistency: &v1beta1.Consistency{
			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: resp.GetConsistencyToken(),
			},
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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	_, err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)
	//&v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "default_viewers"}}
	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject: &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: "u1"}},
		//Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
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

	_, err = seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
	assert.NoError(t, err)
	_, err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
	assert.NoError(t, err)
	_, err = seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u2")
	assert.NoError(t, err)
	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)

	responseCollector := NewLookup_SubjectsServerStub(ctx)
	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
		SubjectType: rbac_ns_type("principal"),
		Relation:    "view",
		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"u1", "u2"}, ids)
}

// Test is amibguous as consistency token may not be *strictly* used.
// if a better revision is available and faster than it will be used, causing
// race conditions for this test to failure
// func TestLookupService_LookupSubjectsMissingItems_WithWrongConsistencyToken(t *testing.T) {
// 	t.Parallel()
// 	ctx := context.TODO()
// 	spicedb, err := container.CreateSpiceDbRepository()
// 	assert.NoError(t, err)

// 	resp1, err := seedWidgetInDefaultWorkspace(ctx, spicedb, "thing1")
// 	assert.NoError(t, err)
// 	resp2, err := seedUserWithViewThingInDefaultWorkspace(ctx, spicedb, "u1")
// 	assert.NoError(t, err)

// 	service := createLookupService(spicedb)

// 	// using first consistency token resp1 we expect missing ids
// 	responseCollector := NewLookup_SubjectsServerStub(ctx)
// 	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
// 		SubjectType: rbac_ns_type("principal"),
// 		Relation:    "view",
// 		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
// 		Consistency: &v1beta1.Consistency{
// 			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
// 				AtLeastAsFresh: resp1.GetConsistencyToken(),
// 			},
// 		},
// 	}, responseCollector)
// 	assert.NoError(t, err)
// 	ids := responseCollector.GetIDs()

// 	assert.ElementsMatch(t, []string{}, ids)

// 	// using latest consistency token resp2 we expect all ids!
// 	responseCollector = NewLookup_SubjectsServerStub(ctx)
// 	err = service.LookupSubjects(&v1beta1.LookupSubjectsRequest{
// 		SubjectType: rbac_ns_type("principal"),
// 		Relation:    "view",
// 		Resource:    &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: "thing1"},
// 		Consistency: &v1beta1.Consistency{
// 			Requirement: &v1beta1.Consistency_AtLeastAsFresh{
// 				AtLeastAsFresh: resp2.GetConsistencyToken(),
// 			},
// 		},
// 	}, responseCollector)
// 	assert.NoError(t, err)
// 	ids = responseCollector.GetIDs()

// 	assert.ElementsMatch(t, []string{"u1"}, ids)
// }

func TestLookupService_LookupResources_IgnoresSubjectRelation(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	memberRelation := "member"
	_, err = spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "rb1"},
			Relation: "subject",
			Subject:  &v1beta1.SubjectReference{Relation: &memberRelation, Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("group"), Id: "g1"}},
		},
	}, biz.TouchSemantics(true))
	assert.NoError(t, err)

	_, err = spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("group"), Id: "g1"},
			Relation: "member",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: "p1"}},
		},
	}, biz.TouchSemantics(true))
	assert.NoError(t, err)

	container.WaitForQuantizationInterval()

	service := createLookupService(spicedb)
	responseCollector := NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: "p1"}},
		Relation: "subject",
		ResourceType: &v1beta1.ObjectType{
			Name:      "role_binding",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	ids := responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"rb1"}, ids)

	responseCollector = NewLookup_ResourcesServerStub(ctx)
	err = service.LookupResources(&v1beta1.LookupResourcesRequest{
		Subject:  &v1beta1.SubjectReference{Relation: &memberRelation, Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("group"), Id: "g1"}},
		Relation: "subject",
		ResourceType: &v1beta1.ObjectType{
			Name:      "role_binding",
			Namespace: "rbac",
		},
	}, responseCollector)
	assert.NoError(t, err)
	ids = responseCollector.GetIDs()

	assert.ElementsMatch(t, []string{"rb1"}, ids)

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
func seedWidgetInDefaultWorkspace(ctx context.Context, spicedb *data.SpiceDbRepository, thing string) (*v1beta1.CreateTuplesResponse, error) {
	return spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("widget"), Id: thing},
			Relation: "workspace",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"}},
		},
	}, biz.TouchSemantics(true))
}

func seedUserWithViewThingInDefaultWorkspace(ctx context.Context, spicedb *data.SpiceDbRepository, user string) (*v1beta1.CreateTuplesResponse, error) {
	return spicedb.CreateRelationships(ctx, []*v1beta1.Relationship{
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("role"), Id: "viewers"},
			Relation: "view_widget",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: "*"}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "default_viewers"},
			Relation: "subject",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: user}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "default_viewers_two"},
			Relation: "subject",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("principal"), Id: user}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "default_viewers"},
			Relation: "granted",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("role"), Id: "viewers"}},
		},
		{
			Resource: &v1beta1.ObjectReference{Type: rbac_ns_type("workspace"), Id: "default"},
			Relation: "user_grant",
			Subject:  &v1beta1.SubjectReference{Subject: &v1beta1.ObjectReference{Type: rbac_ns_type("role_binding"), Id: "default_viewers"}},
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
