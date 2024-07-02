package main

import (
	"flag"
	"os"
	"strings"

	"github.com/go-kratos/kratos/v2/config/env"

	"github.com/project-kessel/relations-api/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()

	c := config.New(
		config.WithSource(
			env.NewSource("SPICEDB_"),
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	//preshared, err := c.Value("PRESHARED").String()
	//if err != nil {
	//	log.NewHelper(logger).Errorf("Failed to read preshared key env %d", err)
	//}
	//if preshared != "" {
	//	bc.Data.SpiceDb.Token = preshared
	//}

	app, cleanup, err := wireApp(bc.Server, bc.Data, createLogger(bc.Server))
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func createLogger(c *conf.Server) log.Logger {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	var level log.Level
	if c.MinLogLevel == nil {
		level = log.LevelInfo
	} else {
		switch strings.ToUpper(*c.MinLogLevel) {
		case "DEBUG":
			level = log.LevelDebug
		case "INFO":
			level = log.LevelInfo
		case "WARN":
			level = log.LevelWarn
		case "ERROR":
			level = log.LevelError
		case "FATAL":
			level = log.LevelFatal
		}
	}
	logger = log.NewFilter(logger, log.FilterLevel(level))

	return logger
}
