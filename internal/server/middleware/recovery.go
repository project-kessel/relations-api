package middleware

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"

	"google.golang.org/grpc"
)

type Option func(*options)

type options struct { // Duplicated from https://github.com/go-kratos/kratos/blob/main/middleware/recovery/recovery.go b/c no export
	handler recovery.HandlerFunc
}

func StreamRecoveryInterceptor(logger log.Logger, opts ...Option) grpc.StreamServerInterceptor {
	op := options{
		handler: func(ctx context.Context, req, err interface{}) error {
			return recovery.ErrUnknownRequest
		},
	}
	for _, o := range opts {
		o(&op)
	}
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		startTime := time.Now()
		wrapper := &requestInterceptingWrapper{ServerStream: ss}
		ctx := ss.Context()

		defer func() {
			if rerr := recover(); rerr != nil {
				buf := make([]byte, 64<<10)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				log.NewHelper(log.WithContext(ctx, logger)).Log(
					log.LevelError,
					"latency", time.Since(startTime).Seconds(),
					"reason", fmt.Sprintf("%v: %+v\n%s\n", rerr, wrapper.req, buf),
				)
				err = op.handler(ctx, wrapper.req, rerr)
			}
		}()
		return handler(srv, wrapper)
	}
}
