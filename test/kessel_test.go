package test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/authzed/grpcutil"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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

	// make initial empty request to load schema for first time use.
	localKesselContainer.spicedbContainer.WaitForQuantizationInterval()
	err = loadSchema()
	if err != nil {
		localKesselContainer.Close()
		panic(fmt.Errorf("Failed to load schema, %w", err))
	}
	// wait a bit before activating tests that will actually use the loaded schema
	localKesselContainer.spicedbContainer.WaitForQuantizationInterval()

	result := m.Run()

	localKesselContainer.Close()
	os.Exit(result)
}

func loadSchema() error {
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselCheckServiceClient(conn)

	// send valid CheckRequest to hit service and load schema.
	_, err = client.Check(context.Background(), &v1beta1.CheckRequest{
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "bob",
			},
		},
		Relation: "member",
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "group",
			},
			Id: "bob_club",
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func TestKesselAPIGRPC_CreateTuples(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}

	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselTupleServiceClient(conn)
	rels := createRelations("principal", "bob", "member", "group", "bob_club")
	_, err = client.CreateTuples(context.Background(), &v1beta1.CreateTuplesRequest{
		Tuples: rels,
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_ReadTuples(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselTupleServiceClient(conn)
	_, err = client.ReadTuples(context.Background(), &v1beta1.ReadTuplesRequest{
		Filter: &v1beta1.RelationTupleFilter{
			ResourceNamespace: pointerize("rbac"),
			ResourceType:      pointerize("group"),
			ResourceId:        pointerize("bob_club"),
			Relation:          pointerize("member"),
			SubjectFilter: &v1beta1.SubjectFilter{
				SubjectNamespace: pointerize("rbac"),
				SubjectType:      pointerize("principal"),
				SubjectId:        pointerize("bob"),
			},
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_DeleteTuples(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselTupleServiceClient(conn)

	_, err = client.DeleteTuples(context.Background(), &v1beta1.DeleteTuplesRequest{
		Filter: &v1beta1.RelationTupleFilter{
			ResourceNamespace: pointerize("rbac"),
			ResourceType:      pointerize("group"),
			ResourceId:        pointerize("bob_club"),
			Relation:          pointerize("member"),
			SubjectFilter: &v1beta1.SubjectFilter{
				SubjectNamespace: pointerize("rbac"),
				SubjectType:      pointerize("principal"),
				SubjectId:        pointerize("bob"),
			},
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_Check(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselCheckServiceClient(conn)

	_, err = client.Check(context.Background(), &v1beta1.CheckRequest{
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "bob",
			},
		},
		Relation: "member",
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "group",
			},
			Id: "bob_club",
		},
	})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_LookupSubjects(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselLookupServiceClient(conn)

	_, err = client.LookupSubjects(
		context.Background(), &v1beta1.LookupSubjectsRequest{
			Resource:    &v1beta1.ObjectReference{Type: simple_type("thing"), Id: "thing1"},
			Relation:    "view",
			SubjectType: simple_type("principal"),
		})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_LookupResources(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselLookupServiceClient(conn)

	_, err = client.LookupResources(
		context.Background(), &v1beta1.LookupResourcesRequest{
			ResourceType: &v1beta1.ObjectType{Name: "group", Namespace: "rbac"},
			Relation:     "member",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: &v1beta1.ObjectType{
						Name:      "principal",
						Namespace: "rbac",
					},
					Id: "bob",
				},
			},
		})
	assert.NoError(t, err)
}

func TestKesselAPIGRPC_LookupResourcesInvalid(t *testing.T) {
	//Ensures that validation middleware is still active with authentication enabled
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	client := v1beta1.NewKesselLookupServiceClient(conn)

	stream, err := client.LookupResources(
		context.Background(), &v1beta1.LookupResourcesRequest{})
	assert.NoError(t, err)

	_, err = stream.Recv() //Errors are returned with the first response, not the initial request

	status, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, status.Code())
}

func TestKesselAPIGRPC_BulkCheck(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	// Create relationships needed for the permission scenario, mirroring internal/data test data
	tupleClient := v1beta1.NewKesselTupleServiceClient(conn)
	var rels []*v1beta1.Relationship
	rels = append(rels, createRelations("role_binding", "rb_test", "user_grant", "workspace", "test")...)
	rels = append(rels, createRelations("role", "rl1", "granted", "role_binding", "rb_test")...)
	rels = append(rels, createRelations("principal", "bob", "subject", "role_binding", "rb_test")...)
	rels = append(rels, createRelations("principal", "*", "view_widget", "role", "rl1")...)

	_, err = tupleClient.CreateTuples(context.Background(), &v1beta1.CreateTuplesRequest{
		Tuples: rels,
	})
	assert.NoError(t, err)

	// Wait for SpiceDB quantization interval so the checks see the new tuples
	localKesselContainer.spicedbContainer.WaitForQuantizationInterval()

	// Prepare a single bulk check request for both bob and alice
	items := []*v1beta1.CheckBulkRequestItem{
		{
			Resource: &v1beta1.ObjectReference{
				Type: simple_type("workspace"),
				Id:   "test",
			},
			Relation: "view_widget",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: simple_type("principal"),
					Id:   "bob",
				},
			},
		},
		{
			Resource: &v1beta1.ObjectReference{
				Type: simple_type("workspace"),
				Id:   "test",
			},
			Relation: "view_widget",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: simple_type("principal"),
					Id:   "alice",
				},
			},
		},
	}
	req := &v1beta1.CheckBulkRequest{Items: items}
	checkClient := v1beta1.NewKesselCheckServiceClient(conn)
	resp, err := checkClient.CheckBulk(context.Background(), req)
	assert.NoError(t, err)
	if !assert.Equal(t, 2, len(resp.GetPairs())) {
		return
	}
	results := map[string]v1beta1.CheckBulkResponseItem_Allowed{}
	for _, p := range resp.GetPairs() {
		subjId := p.GetRequest().GetSubject().GetSubject().GetId()
		results[subjId] = p.GetItem().GetAllowed()
	}
	assert.Equal(t, v1beta1.CheckBulkResponseItem_ALLOWED_TRUE, results["bob"])
	assert.Equal(t, v1beta1.CheckBulkResponseItem_ALLOWED_FALSE, results["alice"])
}

func TestKesselAPIGRPC_BulkCheck_WithErrorPair(t *testing.T) {
	t.Parallel()
	kcurl := fmt.Sprintf("http://localhost:%s", localKesselContainer.kccontainer.GetPort("8080/tcp"))
	token, err := GetJWTToken(kcurl, "admin", "admin")
	if err != nil {
		fmt.Print(err)
	}
	conn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%s", localKesselContainer.gRPCport),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(token.AccessToken),
	)
	if err != nil {
		fmt.Print(err)
	}

	// Seed relationships for valid checks
	tupleClient := v1beta1.NewKesselTupleServiceClient(conn)
	var rels []*v1beta1.Relationship
	rels = append(rels, createRelations("role_binding", "rb_test2", "user_grant", "workspace", "test2")...)
	rels = append(rels, createRelations("role", "rl2", "granted", "role_binding", "rb_test2")...)
	rels = append(rels, createRelations("principal", "bob", "subject", "role_binding", "rb_test2")...)
	rels = append(rels, createRelations("principal", "*", "view_widget", "role", "rl2")...)

	_, err = tupleClient.CreateTuples(context.Background(), &v1beta1.CreateTuplesRequest{
		Tuples: rels,
	})
	assert.NoError(t, err)

	// Wait for SpiceDB quantization interval so the checks see the new tuples
	localKesselContainer.spicedbContainer.WaitForQuantizationInterval()

	// Prepare a request with:
	// - one allowed TRUE (bob)
	// - one allowed FALSE (alice)
	// - one item that should produce an internal per-item error (invalid subject type),
	//   which the API maps to ALLOWED_UNSPECIFIED in the response pair.
	items := []*v1beta1.CheckBulkRequestItem{
		{
			Resource: &v1beta1.ObjectReference{
				Type: simple_type("workspace"),
				Id:   "test2",
			},
			Relation: "view_widget",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: simple_type("principal"),
					Id:   "bob",
				},
			},
		},
		{
			Resource: &v1beta1.ObjectReference{
				Type: simple_type("workspace"),
				Id:   "test2",
			},
			Relation: "view_widget",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: simple_type("principal"),
					Id:   "alice",
				},
			},
		},
		{
			// invalid subject type name triggers SpiceDB per-item error; server maps to ALLOWED_UNSPECIFIED
			Resource: &v1beta1.ObjectReference{
				Type: simple_type("workspace"),
				Id:   "test2",
			},
			Relation: "view_widget",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: simple_type("not_a_user"),
					Id:   "charlie",
				},
			},
		},
	}
	req := &v1beta1.CheckBulkRequest{Items: items}
	checkClient := v1beta1.NewKesselCheckServiceClient(conn)
	resp, err := checkClient.CheckBulk(context.Background(), req)
	assert.NoError(t, err)
	if !assert.Equal(t, 3, len(resp.GetPairs())) {
		return
	}
	results := map[string]v1beta1.CheckBulkResponseItem_Allowed{}
	for _, p := range resp.GetPairs() {
		subjId := p.GetRequest().GetSubject().GetSubject().GetId()
		results[subjId] = p.GetItem().GetAllowed()
	}
	assert.Equal(t, v1beta1.CheckBulkResponseItem_ALLOWED_TRUE, results["bob"])
	assert.Equal(t, v1beta1.CheckBulkResponseItem_ALLOWED_FALSE, results["alice"])
	assert.Equal(t, v1beta1.CheckBulkResponseItem_ALLOWED_UNSPECIFIED, results["charlie"])
}

func pointerize(value string) *string { //Used to turn string literals into pointers
	return &value
}

func simple_type(typename string) *v1beta1.ObjectType {
	return &v1beta1.ObjectType{Name: typename, Namespace: "rbac"}
}

func createRelations(subName string, subId string, relation string, resouceName string, ResouceId string) []*v1beta1.Relationship {
	rels := []*v1beta1.Relationship{
		{
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: &v1beta1.ObjectType{
						Name:      subName,
						Namespace: "rbac",
					},
					Id: subId,
				},
			},
			Relation: relation,
			Resource: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Name:      resouceName,
					Namespace: "rbac",
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
