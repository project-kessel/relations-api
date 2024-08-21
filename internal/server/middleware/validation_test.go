package middleware

import (
	"context"
	"testing"

	"github.com/bufbuild/protovalidate-go"
	"github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestValidationMiddleware_ValidRequest(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	assert.NoError(t, err)

	m := ValidationMiddleware(validator)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := m(handler)(context.Background(), &v1beta1.LookupSubjectsRequest{
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{Namespace: "rbac", Name: "user"},
			Id:   "bob"},
		Relation:    "member",
		SubjectType: &v1beta1.ObjectType{Namespace: "rbac", Name: "group"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestValidationMiddleware_InvalidRequest(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	assert.NoError(t, err)

	m := ValidationMiddleware(validator)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	resp, err := m(handler)(context.Background(), &v1beta1.LookupSubjectsRequest{
		Resource: &v1beta1.ObjectReference{},
	})
	assert.Error(t, err)
	assert.Equal(t, resp, nil)
}
