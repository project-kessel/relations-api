package data

import (
	apiV1 "ciam-rebac/api/rebac/v1"
	"ciam-rebac/internal/biz"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
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

var operations = map[string]map[string][]string{
	"inventory": {
		"all":    {"read", "all"},
		"hosts":  {"read", "write"},
		"groups": {"read", "write", "all"},
	},
	"cost-management": {},
}

const (
	DefaultWorkspace string = "aspian/default"
	WorkspaceA       string = "aspian/default/A"
	WorkspaceB       string = "aspian/default/B"
	User             string = "user"
)

func seedRoleTestingRelationships(spiceDb *SpiceDbRepository) {
	createRelationship(DefaultWorkspace, "workspace", "", "parent", "workspace", WorkspaceA)
	createRelationship(DefaultWorkspace, "workspace", "", "parent", "workspace", WorkspaceB)
}

func TestSingleGlobalPermissionRoleConversion(t *testing.T) {
	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()

	assert.NoError(t, err)

	role := V1Role{
		Id:                  "4cc8ea10-dfe3-11ee-8af7-6777abd80b44",
		Name:                "Inventory Host Readers",
		GlobalPermissions:   []string{"inventory:hosts:read"},
		FilteredPermissions: map[string]map[string]AttributeFilter{},
	}

	seedRoleTestingRelationships(spiceDbRepo)

	assertRole(t, ctx, spiceDbRepo, User, role)
}

func assertRole(t *testing.T, ctx context.Context, spiceDb *SpiceDbRepository, user string, role V1Role) bool {
	for appName, app := range operations {
		for resourceType, operations := range app {
			for _, operation := range operations {
				//Check global permission
				shouldHave := contains(operation, role.GlobalPermissions)
				has := checkGlobalPermission(t, ctx, spiceDb, user, appName+"_"+resourceType+"_"+operation) //This probably works- maybe try writing a test that leverages just it?
				assert.Equal(t, shouldHave, has, "User: %s, permission: %s, should have: %b, has: %b", user, operation, shouldHave, has)

				//Check attribute filters
				if filter, found := role.FilteredPermissions[appName][resourceType]; found {
					if has {
						t.Log("Cannot check filtered permission when the permission is granted globally")
						continue
					}
					//Potential problem! Wildcards will cause false positives!
					//Potential solution: have the test case specify what the results should be separately from what the permissions should be. These could go on the AttributeFilter struct
					//	This might be the way, if it's efficient and based on well-known values. It lets the test author be explicit about what's expected.
					//Potential solution: have the operations data be structs that include pointing to the ones they should imply
					//	This seems really efficient by its nature but there's a catch- it's not clear to the test author what's expected, and it's possible the test could be comparing two flawed sets under the hood

					//Related potential problem: we sometimes use the full permission name, like inventory:hosts:read and sometimes short names like read
					authorizedIDs := getAuthorizedIDs(t, ctx, spiceDb, user, resourceType, operation)
					assert.ElementsMatch(t, filter.ExpectedResourceIDs, authorizedIDs)
				}
			}
		}
	}

	return false
}

func contains(find string, values []string) bool {
	for _, value := range values {
		if find == value {
			return true
		}
	}

	return false
}

func checkGlobalPermission(t *testing.T, ctx context.Context, spiceDb *SpiceDbRepository, user string, operation string) bool {
	result, err := spiceDb.client.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		Resource: &v1.ObjectReference{
			ObjectType: "workspace",
			ObjectId:   "default",
		},
		Permission: operation,
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: "user",
				ObjectId:   user,
			},
		},
	})

	assert.NoError(t, err, "Checking permission")

	return result.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
}

func getAuthorizedIDs(t *testing.T, ctx context.Context, spiceDb *SpiceDbRepository, user string, resourceType string, operation string) []string {
	result, err := spiceDb.client.LookupResources(ctx, &v1.LookupResourcesRequest{
		Consistency: &v1.Consistency{
			Requirement: &v1.Consistency_FullyConsistent{
				FullyConsistent: true,
			},
		},
		ResourceObjectType: resourceType,
		Permission:         operation,
		Subject: &v1.SubjectReference{
			Object: &v1.ObjectReference{
				ObjectType: "user",
				ObjectId:   user,
			},
		},
	})

	assert.NoError(t, err, "Looking up resources")

	authorizedIDs := make([]string, 0)

	resp, err := result.Recv()
	for err == nil {
		authorizedIDs = append(authorizedIDs, resp.GetResourceObjectId())
		resp, err = result.Recv()
	}

	assert.ErrorIs(t, err, io.EOF, "Error streaming resources")

	return authorizedIDs
}

type V1Role struct {
	Id                  string
	Name                string
	GlobalPermissions   []string
	FilteredPermissions map[string]map[string]AttributeFilter
}

type AttributeFilter struct {
	ExplicitResourceIDs []string
	ExpectedResourceIDs []string
}
