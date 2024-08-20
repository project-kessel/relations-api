package data

import (
	"context"
	"fmt"
	"os"
	"testing"

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

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
	assert.True(t, exists)
}

func TestSecondCreateRelationshipFailsWithTouchFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.AlreadyExists)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
	assert.True(t, exists)
}

func TestSecondCreateRelationshipSucceedsWithTouchTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	touch = true

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "rbac/user", "", "member", "rbac/group", "bob_club")
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

func TestCreateRelationshipFailsWithBadSubjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badSubjectType := simple_type("not_a_user")

	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", badSubjectType, "", "member", simple_type("group"), "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.FailedPrecondition)
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", badSubjectType.GetNamespace(), badSubjectType.GetName()))
}

func TestCreateRelationshipFailsWithBadObjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badObjectType := simple_type("not_an_object")

	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", simple_type("user"), "", "member", badObjectType, "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.FailedPrecondition)
	assert.Contains(t, err.Error(),
		fmt.Sprintf("object definition `%s/%s` not found", badObjectType.GetNamespace(), badObjectType.GetName()))

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
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	readRelChan, _, err := spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("bob_club"),
		ResourceType: pointerize("rbac/group"),
		Relation:     pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("rbac/user"),
		},
	}, 0, "")

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
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	readRelChan, _, err := spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("bob_club"),
		ResourceType: pointerize("rbac/group"),
		Relation:     pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("rbac/user"),
		},
	}, 0, "")

	if !assert.NoError(t, err) {
		return
	}

	readrels := spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 1, len(readrels))

	err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("bob_club"),
		ResourceType: pointerize("rbac/group"),
		Relation:     pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("rbac/user"),
		},
	})

	if !assert.NoError(t, err) {
		return
	}

	readRelChan, _, err = spiceDbRepo.ReadRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("bob_club"),
		ResourceType: pointerize("rbac/group"),
		Relation:     pointerize("member"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("rbac/user"),
		},
	}, 0, "")

	if !assert.NoError(t, err) {
		return
	}

	readrels = spiceRelChanToSlice(readRelChan)
	assert.Equal(t, 0, len(readrels))

}

func TestSpiceDbRepository_CheckPermission(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	//group:bob_club#member@user:bob
	//workspace:test#user_grant@role_binding:rb_test
	//role_binding:rb_test#granted@role:rl1
	//role_binding:rb_test#subject@user:bob
	//role:rl1#view_the_thing@user:*
	rels := []*apiV1beta1.Relationship{
		createRelationship("bob", simple_type("user"), "", "member", simple_type("group"), "bob_club"),
		createRelationship("rb_test", simple_type("role_binding"), "", "user_grant", simple_type("workspace"), "test"),
		createRelationship("rl1", simple_type("role"), "", "granted", simple_type("role_binding"), "rb_test"),
		createRelationship("bob", simple_type("user"), "", "subject", simple_type("role_binding"), "rb_test"),
		createRelationship("*", simple_type("user"), "", "view_the_thing", simple_type("role"), "rl1"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: simple_type("user"),
			Id:   "bob",
		},
	}

	resource := &apiV1beta1.ObjectReference{
		Type: simple_type("workspace"),
		Id:   "test",
	}
	// zed permission check workspace:test view_the_thing user:bob --explain
	check := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_the_thing",
		Resource: resource,
	}
	resp, err := spiceDbRepo.Check(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse := apiV1beta1.CheckResponse{
		Allowed: apiV1beta1.CheckResponse_ALLOWED_TRUE,
	}
	assert.Equal(t, &checkResponse, resp)

	//Remove // role_binding:rb_test#subject@user:bob
	err = spiceDbRepo.DeleteRelationships(ctx, &apiV1beta1.RelationTupleFilter{
		ResourceId:   pointerize("rb_test"),
		ResourceType: pointerize("rbac/role_binding"),
		Relation:     pointerize("subject"),
		SubjectFilter: &apiV1beta1.SubjectFilter{
			SubjectId:   pointerize("bob"),
			SubjectType: pointerize("rbac/user"),
		},
	})
	if !assert.NoError(t, err) {
		return
	}

	// zed permission check workspace:test view_the_thing user:bob --explain
	check2 := apiV1beta1.CheckRequest{
		Subject:  subject,
		Relation: "view_the_thing",
		Resource: resource,
	}

	resp2, err := spiceDbRepo.Check(ctx, &check2)
	if !assert.NoError(t, err) {
		return
	}
	checkResponsev2 := apiV1beta1.CheckResponse{
		Allowed: apiV1beta1.CheckResponse_ALLOWED_FALSE,
	}
	assert.Equal(t, &checkResponsev2, resp2)
}

func simple_type(typename string) *apiV1beta1.ObjectType {
	return &apiV1beta1.ObjectType{Name: typename, Namespace: "rbac"}
}

func pointerize(value string) *string { //Used to turn string literals into pointers
	return &value
}

func createRelationship(subjectId string, subjectType *apiV1beta1.ObjectType, subjectRelationship string, relationship string, objectType *apiV1beta1.ObjectType, objectId string) *apiV1beta1.Relationship {
	subject := &apiV1beta1.SubjectReference{
		Subject: &apiV1beta1.ObjectReference{
			Type: subjectType,
			Id:   subjectId,
		},
	}

	if subjectRelationship != "" {
		subject.Relation = &subjectRelationship
	}

	resource := &apiV1beta1.ObjectReference{
		Type: objectType,
		Id:   objectId,
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
