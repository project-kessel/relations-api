package middleware

import (
	"context"
	"testing"

	"github.com/bufbuild/protovalidate-go"
	"github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
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
			Id:   "bob",
		},
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

type DummyServerStream struct {
	grpc.ServerStream
	RecvMsgFunc func(msg interface{}) error
}

func (m *DummyServerStream) RecvMsg(msg interface{}) error {
	return m.RecvMsgFunc(msg)
}

func TestStreamValidationInterceptor_ValidRequest(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	assert.NoError(t, err)

	interceptor := StreamValidationInterceptor(validator)

	dummyStream := &DummyServerStream{
		RecvMsgFunc: func(msg interface{}) error {
			*msg.(*v1beta1.LookupSubjectsRequest) = v1beta1.LookupSubjectsRequest{
				Resource: &v1beta1.ObjectReference{
					Type: &v1beta1.ObjectType{Namespace: "rbac", Name: "user"},
					Id:   "bob",
				},
				Relation:    "member",
				SubjectType: &v1beta1.ObjectType{Namespace: "rbac", Name: "group"},
			}
			return nil
		},
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		msg := &v1beta1.LookupSubjectsRequest{}
		err := stream.RecvMsg(msg)
		return err
	}

	err = interceptor(nil, dummyStream, nil, handler)
	assert.NoError(t, err)
}

func TestStreamValidationInterceptor_InvalidRequest(t *testing.T) {
	t.Parallel()

	validator, err := protovalidate.New()
	assert.NoError(t, err)

	interceptor := StreamValidationInterceptor(validator)

	dummyStream := &DummyServerStream{
		RecvMsgFunc: func(msg interface{}) error {
			*msg.(*v1beta1.LookupSubjectsRequest) = v1beta1.LookupSubjectsRequest{
				Resource: &v1beta1.ObjectReference{},
			}
			return nil
		},
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		msg := &v1beta1.LookupSubjectsRequest{}
		err := stream.RecvMsg(msg)
		return err
	}

	err = interceptor(nil, dummyStream, nil, handler)
	assert.Error(t, err)
}
