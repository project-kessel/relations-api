package data

import (
	apiV1 "ciam-rebac/api/rebac/v1"
	"ciam-rebac/internal/biz"
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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

func TestWriteRelationship(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	spiceDbRepo, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	preExisting := container.CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.False(t, preExisting)

	rels := []*apiV1.Relationship{
		createRelationship("bob", "user", "", "member", "group", "bob_club"),
	}

	semantics := biz.TouchSemantics(false)

	err = spiceDbRepo.CreateRelationships(ctx, rels, semantics)
	assert.NoError(t, err)

	exists := container.CheckForRelationship(spiceDbRepo.client, "bob", "user", "", "member", "group", "bob_club")
	assert.True(t, exists)
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
