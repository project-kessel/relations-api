package middleware

import (
	"context"

	"github.com/bufbuild/protovalidate-go"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func StreamValidationInterceptor(validator *protovalidate.Validator) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &requestValidatingWrapper{ServerStream: ss, Validator: validator}
		return handler(srv, wrapper)
	}
}

type requestValidatingWrapper struct {
	grpc.ServerStream
	*protovalidate.Validator
}

func (w *requestValidatingWrapper) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	if v, ok := m.(proto.Message); ok {
		if err = w.Validator.Validate(v); err != nil {
			return errors.BadRequest("VALIDATOR", err.Error()).WithCause(err)
		}
	}

	return nil
}

func (w *requestValidatingWrapper) SendMsg(m interface{}) error {
	return w.ServerStream.SendMsg(m)
}

func ValidationMiddleware(validator *protovalidate.Validator) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if v, ok := req.(proto.Message); ok {
				if err := validator.Validate(v); err != nil {
					return nil, errors.BadRequest("VALIDATOR", err.Error()).WithCause(err)
				}
			}
			return handler(ctx, req)
		}
	}
}
