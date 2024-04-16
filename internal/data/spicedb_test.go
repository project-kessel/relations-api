package data

import (
	apiV1 "ciam-rebac/api/rebac/v1"
	"ciam-rebac/internal/biz"
	"context"
	"fmt"
	"os"
	"testing"

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

	container, err = CreateContainer(logger)

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

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.True(t, exists)
}

func TestSecondCreateRelationshipFailsWithTouchFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.AlreadyExists)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.True(t, exists)
}

func TestSecondCreateRelationshipSucceedsWithTouchTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	touch = true

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.NoError(t, err)

	exists := CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.True(t, exists)
}

func TestCreateRelationshipFailsWithBadSubjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badSubjectType := "not_a_user"

	rels := []*apiV1.Relationship{
		createRelationship("bob", badSubjectType, "", "member", "group", "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.FailedPrecondition)
	assert.Contains(t, err.Error(), "object definition `"+badSubjectType+"` not found")
}

func TestCreateRelationshipFailsWithBadObjectType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	badObjectType := "not_an_object"

	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", badObjectType, "bob_club"),
	}

	touch := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, touch)
	assert.Error(t, err)
	assert.Equal(t, status.Convert(err).Code(), codes.FailedPrecondition)
	assert.Contains(t, err.Error(), "object definition `"+badObjectType+"` not found")
}

func TestWriteAndReadBackRelationships(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)
	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	readrels, err := spiceDbRepo.ReadRelationships(ctx, &apiV1.RelationshipFilter{
		ObjectId:   "bob_club",
		ObjectType: "group",
		Relation:   "member",
		SubjectFilter: &apiV1.SubjectFilter{
			SubjectId:   "bob",
			SubjectType: "user",
		},
	})

	if !assert.NoError(t, err) {
		return
	}

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
	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	readrels, err := spiceDbRepo.ReadRelationships(ctx, &apiV1.RelationshipFilter{
		ObjectId:   "bob_club",
		ObjectType: "group",
		Relation:   "member",
		SubjectFilter: &apiV1.SubjectFilter{
			SubjectId:   "bob",
			SubjectType: "user",
		},
	})

	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, 1, len(readrels))

	err = spiceDbRepo.DeleteRelationships(ctx, &apiV1.RelationshipFilter{
		ObjectId:   "bob_club",
		ObjectType: "group",
		Relation:   "member",
		SubjectFilter: &apiV1.SubjectFilter{
			SubjectId:   "bob",
			SubjectType: "user",
		},
	})

	if !assert.NoError(t, err) {
		return
	}

	readrels, err = spiceDbRepo.ReadRelationships(ctx, &apiV1.RelationshipFilter{
		ObjectId:   "bob_club",
		ObjectType: "group",
		Relation:   "member",
		SubjectFilter: &apiV1.SubjectFilter{
			SubjectId:   "bob",
			SubjectType: "user",
		},
	})

	if !assert.NoError(t, err) {
		return
	}

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
	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
		createRelationship("rb_test", "role_binding", "", "user_grant", "workspace", "test"),
		createRelationship("rl1", "role", "", "granted", "role_binding", "rb_test"),
		createRelationship("bob", "user", "", "subject", "role_binding", "rb_test"),
		createRelationship("*", "user", "", "view_the_thing", "role", "rl1"),
	}

	err = spiceDbRepo.CreateRelationships(ctx, rels, biz.TouchSemantics(true))
	if !assert.NoError(t, err) {
		return
	}

	subject := &apiV1.SubjectReference{
		Object: &apiV1.ObjectReference{
			Type: "user",
			Id:   "bob",
		},
	}

	object := &apiV1.ObjectReference{
		Type: "workspace",
		Id:   "test",
	}
	// zed permission check workspace:test view_the_thing user:bob --explain
	check := apiV1.CheckRequest{
		Subject:  subject,
		Relation: "view_the_thing",
		Object:   object,
	}
	resp, err := spiceDbRepo.Check(ctx, &check)
	if !assert.NoError(t, err) {
		return
	}
	//apiV1.CheckResponse_ALLOWED_TRUE
	checkResponse := apiV1.CheckResponse{
		Allowed: apiV1.CheckResponse_ALLOWED_TRUE,
	}
	assert.Equal(t, &checkResponse, resp)

	//Remove // role_binding:rb_test#subject@user:bob
	err = spiceDbRepo.DeleteRelationships(ctx, &apiV1.RelationshipFilter{
		ObjectId:   "rb_test",
		ObjectType: "role_binding",
		Relation:   "subject",
		SubjectFilter: &apiV1.SubjectFilter{
			SubjectId:   "bob",
			SubjectType: "user",
		},
	})
	if !assert.NoError(t, err) {
		return
	}

	// zed permission check workspace:test view_the_thing user:bob --explain
	check2 := apiV1.CheckRequest{
		Subject:  subject,
		Relation: "view_the_thing",
		Object:   object,
	}

	resp2, err := spiceDbRepo.Check(ctx, &check2)
	if !assert.NoError(t, err) {
		return
	}
	checkResponsev2 := apiV1.CheckResponse{
		Allowed: apiV1.CheckResponse_ALLOWED_FALSE,
	}
	assert.Equal(t, &checkResponsev2, resp2)
}
func createRelationship(subjectId string, subjectType string, subjectRelationship string, relationship string, objectType string, objectId string) *apiV1.Relationship {
	subject := &apiV1.SubjectReference{
		Object: &apiV1.ObjectReference{
			Type: subjectType,
			Id:   subjectId,
		},
		Relation: subjectRelationship,
	}

	object := &apiV1.ObjectReference{
		Type: objectType,
		Id:   objectId,
	}

	return &apiV1.Relationship{
		Object:   object,
		Relation: relationship,
		Subject:  subject,
	}
}
