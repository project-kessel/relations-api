package service

import (
	"context"

	pb "github.com/project-kessel/relations-api/api/health/v1"
	"github.com/project-kessel/relations-api/internal/biz"
)

type HealthService struct {
	pb.UnimplementedKesselHealthServer
	backendUseCase *biz.IsBackendAvaliableUsecase
}

func NewHealthService(backendUsecase *biz.IsBackendAvaliableUsecase) *HealthService {
	return &HealthService{
		backendUseCase: backendUsecase,
	}
}

func (s *HealthService) GetLivez(ctx context.Context, req *pb.GetLivezRequest) (*pb.GetLivezReply, error) {
	return &pb.GetLivezReply{Status: "OK", Code: 200}, nil
}

func (s *HealthService) GetReadyz(ctx context.Context, req *pb.GetReadyzRequest) (*pb.GetReadyzReply, error) {
	err := s.backendUseCase.IsBackendAvailable()
	if err != nil {
		return &pb.GetReadyzReply{Status: "Unavailable", Code: 503}, nil
	}
	return &pb.GetReadyzReply{Status: "OK", Code: 200}, nil
}
