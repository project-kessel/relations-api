package service

import (
	"context"

	pb "github.com/project-kessel/relations-api/api/health/v1"
)

type HealthService struct {
	pb.UnimplementedKesselHealthServer
}

func NewHealthService() *HealthService {
	return &HealthService{}
}

func (s *HealthService) GetLivez(ctx context.Context, req *pb.GetLivezRequest) (*pb.GetLivezReply, error) {
	return &pb.GetLivezReply{}, nil
}
func (s *HealthService) GetReadyz(ctx context.Context, req *pb.GetReadyzRequest) (*pb.GetReadyzReply, error) {
	return &pb.GetReadyzReply{}, nil
}
