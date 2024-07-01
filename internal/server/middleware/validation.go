package middleware

import (
	"github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc"
)

type validator interface { //Duplicated from github.com/go-kratos/kratos/v2/middleware/validate because not exported
	Validate() error
}

func StreamValidationInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &requestValidatingWrapper{ServerStream: ss}
		return handler(srv, wrapper)
	}
}

type requestValidatingWrapper struct {
	grpc.ServerStream
}

func (w *requestValidatingWrapper) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}

	if v, ok := m.(validator); ok {
		if err = v.Validate(); err != nil {
			return errors.BadRequest("VALIDATOR", err.Error()).WithCause(err)
		}
	}

	return nil
}

func (w *requestValidatingWrapper) SendMsg(m interface{}) error {
	return w.ServerStream.SendMsg(m)
}
