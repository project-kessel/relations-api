package server

import (
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"google.golang.org/grpc"
)

func StreamLogInterceptor(logger log.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		var (
			code      int32
			reason    string
			kind      string
			operation string
		)
		ctx := ss.Context()
		startTime := time.Now()
		operation = info.FullMethod
		kind = "server"
		wrapper := &requestInterceptingWrapper{ServerStream: ss}
		err := handler(srv, wrapper)

		if se := errors.FromError(err); se != nil {
			code = se.Code
			reason = se.Reason
		}
		level, stack := extractError(err)

		log.NewHelper(log.WithContext(ctx, logger)).Log(level,
			"kind", kind,
			"component", kind,
			"operation", operation,
			"args", extractArgs(wrapper.req),
			"code", code,
			"reason", reason,
			"stack", stack,
			"latency", time.Since(startTime).Seconds())

		return err
	}
}

type requestInterceptingWrapper struct {
	req any
	grpc.ServerStream
}

func (w *requestInterceptingWrapper) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m) //Includes deserializing m, all fields are empty before this point
	if w.req == nil {
		w.req = m
	}

	return err
}

func (w *requestInterceptingWrapper) SendMsg(m interface{}) error {
	return w.ServerStream.SendMsg(m)
}

// Taken from Kratos logging middleware
// extractArgs returns the string of the req
func extractArgs(req interface{}) string {
	if redacter, ok := req.(logging.Redacter); ok {
		return redacter.Redact()
	}
	if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}

// extractError returns the string of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}
