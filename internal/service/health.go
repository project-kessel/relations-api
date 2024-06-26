package service

import (
	"context"

	pb "github.com/project-kessel/relations-api/api/health/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type HealthService struct {
	pb.UnimplementedKesselHealthServer
}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) GetLivez(ctx context.Context, req *pb.GetLivezRequest) (*pb.GetLivezReply, error) {
	return &pb.GetLivezReply{Status: "OK", Code: 200}, nil
}
func (s *HealthService) GetReadyz(ctx context.Context, req *pb.GetReadyzRequest) (*pb.GetReadyzReply, error) {
	check := checkSpiceDBReadyz()
	if check {
		return &pb.GetReadyzReply{Status: "OK", Code: 200}, nil
	}
	return &pb.GetReadyzReply{Status: "Unavailable", Code: 503}, nil
}

func checkSpiceDBReadyz() bool {
	conn, err := grpc.NewClient(
		"spicedb:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false
	}

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return false
	}

	return resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
}
