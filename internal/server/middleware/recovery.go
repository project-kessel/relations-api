package middleware

import (
	"fmt"
	"runtime"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"google.golang.org/grpc"
)

func StreamRecoveryInterceptor(logger log.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &requestInterceptingWrapper{ServerStream: ss}
		startTime := time.Now()
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
				// fail silently?
			}
		}()
		return handler(srv, wrapper)
	}
}
