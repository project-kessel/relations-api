package server

import (
	h "ciam-rebac/api/health/v1"
	v0 "ciam-rebac/api/relations/v0"
	"ciam-rebac/internal/conf"
	"ciam-rebac/internal/service"
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	nethttp "net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, relationships *service.RelationshipsService, health *service.HealthService, check *service.CheckService, subjects *service.LookupService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	if c.Http.Pathprefix != "" {
		opts = append(opts, http.PathPrefix(c.Http.Pathprefix))
	}

	srv := http.NewServer(opts...)
	v0.RegisterKesselTupleServiceHTTPServer(srv, relationships)
	v0.RegisterKesselCheckServiceHTTPServer(srv, check)
	h.RegisterKesselHealthHTTPServer(srv, health)

	srv.HandleFunc("/v0/subjects", func(writer nethttp.ResponseWriter, request *nethttp.Request) {
		conn, err := grpc.NewClient(c.Grpc.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("could not connect to grpc server: %v", err)
		}
		defer conn.Close()

		lookupServiceClient := v0.NewKesselLookupServiceClient(conn)

		body, err := io.ReadAll(request.Body)
		if err != nil {
			nethttp.Error(writer, "Failed to read lookup subject body: "+err.Error(), nethttp.StatusInternalServerError)
			return
		}

		lookupSubjectsRequest := v0.LookupSubjectsRequest{}
		if err := json.Unmarshal(body, &lookupSubjectsRequest); err != nil {
			nethttp.Error(writer, "Failed to unmarshal lookup request body: "+err.Error(), nethttp.StatusBadRequest)
			return
		}
		stream, err := lookupServiceClient.LookupSubjects(context.Background(), &lookupSubjectsRequest)
		if err != nil {
			nethttp.Error(writer, "error grpc stream", nethttp.StatusInternalServerError)
			return
		}
		var responses []*v0.LookupSubjectsResponse
		for {
			response, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				nethttp.Error(writer, "Failed to receive data from lookup subject stream: "+err.Error(), nethttp.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			writer.Header().Set("Transfer-Encoding", "chunked")

			responses = append(responses, response)
		}
		if err := json.NewEncoder(writer).Encode(responses); err != nil {
			nethttp.Error(writer, "Failed to encode lookup subject response: "+err.Error(), nethttp.StatusInternalServerError)
			return
		}
	})
	return srv
}
