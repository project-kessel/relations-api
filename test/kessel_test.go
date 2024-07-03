package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	v0 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var localKesselContainer *LocalKesselContainer

func TestMain(m *testing.M) {
	var err error
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)

	localKesselContainer, err = CreateKesselAPIContainer(logger)
	if err != nil {
		fmt.Printf("Error initializing Docker localKesselContainer: %s", err)
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(p string) {
		err := waitForServiceToBeReady(p)
		if err != nil {
			localKesselContainer.Close()
			panic(fmt.Errorf("Error waiting for Kessel Relations to start: %w", err))
		}
		wg.Done()
	}(localKesselContainer.gRPCport)

	wg.Add(1)
	go func(p string) {
		err := waitForServiceToBeReady(p)
		if err != nil {
			localKesselContainer.Close()
			panic(fmt.Errorf("Error waiting for SpiceDB to start: %w", err))
		}
		wg.Done()
	}(localKesselContainer.spicedbContainer.Port())

	wg.Wait()

	result := m.Run()

	localKesselContainer.Close()
	os.Exit(result)
}

func TestKesselAPIGRPC_CreateTuples(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselTupleServiceClient(conn)
	rels := createRelations("user", "bob", "member", "group", "bob_club")
	_, err = client.CreateTuples(context.Background(), &v0.CreateTuplesRequest{
		Tuples: rels,
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_ReadTuples(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselTupleServiceClient(conn)
	_, err = client.ReadTuples(context.Background(), &v0.ReadTuplesRequest{
		Filter: &v0.RelationTupleFilter{
			ResourceType: pointerize("group"),
			ResourceId:   pointerize("bob_club"),
			Relation:     pointerize("member"),
			SubjectFilter: &v0.SubjectFilter{
				SubjectType: pointerize("user"),
				SubjectId:   pointerize("bob"),
			},
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_DeleteTuples(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselTupleServiceClient(conn)

	_, err = client.DeleteTuples(context.Background(), &v0.DeleteTuplesRequest{
		Filter: &v0.RelationTupleFilter{
			ResourceType: pointerize("group"),
			ResourceId:   pointerize("bob_club"),
			Relation:     pointerize("member"),
			SubjectFilter: &v0.SubjectFilter{
				SubjectType: pointerize("user"),
				SubjectId:   pointerize("bob"),
			},
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_Check(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselCheckServiceClient(conn)

	_, err = client.Check(context.Background(), &v0.CheckRequest{
		Subject: &v0.SubjectReference{
			Subject: &v0.ObjectReference{
				Type: &v0.ObjectType{
					Name: "user",
				},
				Id: "bob",
			},
		},
		Relation: "member",
		Resource: &v0.ObjectReference{
			Type: &v0.ObjectType{
				Name: "group",
			},
			Id: "bob_club",
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_LookupSubjects(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselLookupServiceClient(conn)

	_, err = client.LookupSubjects(
		context.Background(), &v0.LookupSubjectsRequest{
			Resource:    &v0.ObjectReference{Type: simple_type("thing"), Id: "thing1"},
			Relation:    "view",
			SubjectType: simple_type("user"),
		})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_LookupResources(t *testing.T) {
	t.Parallel()
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v0.NewKesselLookupServiceClient(conn)

	_, err = client.LookupResources(
		context.Background(), &v0.LookupResourcesRequest{
			ResourceType: &v0.ObjectType{Name: "group"},
			Relation:     "member",
			Subject: &v0.SubjectReference{
				Subject: &v0.ObjectReference{
					Type: &v0.ObjectType{
						Name: "user",
					},
					Id: "bob",
				},
			},
		})
	assert.NoError(t, err)
}

func pointerize(value string) *string { //Used to turn string literals into pointers
	return &value
}

func simple_type(typename string) *v0.ObjectType {
	return &v0.ObjectType{Name: typename}
}

func createRelations(subName string, subId string, relation string, resouceName string, ResouceId string) []*v0.Relationship {
	rels := []*v0.Relationship{
		{
			Subject: &v0.SubjectReference{
				Subject: &v0.ObjectReference{
					Type: &v0.ObjectType{
						Name: subName,
					},
					Id: subId,
				},
			},
			Relation: relation,
			Resource: &v0.ObjectReference{
				Type: &v0.ObjectType{
					Name: resouceName,
				},
				Id: ResouceId,
			},
		},
	}
	return rels
}

func waitForServiceToBeReady(port string) error {
	address := fmt.Sprintf("localhost:%s", port)
	limit := 30
	wait := 250 * time.Millisecond
	started := time.Now()

	for i := 0; i < limit; i++ {
		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			time.Sleep(wait)
			continue
		}
		client := grpc_health_v1.NewHealthClient(conn)
		resp, err := client.Check(context.TODO(), &grpc_health_v1.HealthCheckRequest{})
		if err != nil {
			time.Sleep(wait)
			continue
		}

		switch resp.Status {
		case grpc_health_v1.HealthCheckResponse_NOT_SERVING, grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN:
			time.Sleep(wait)
			continue
		case grpc_health_v1.HealthCheckResponse_SERVING:
			return nil
		}
	}

	return fmt.Errorf("the health endpoint didn't respond successfully within %f seconds.", time.Since(started).Seconds())
}
