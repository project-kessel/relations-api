package service

import (
	"context"
	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	"github.com/project-kessel/relations-api/internal/biz"
)

type HealthService struct {
	pb.UnimplementedKesselHealthServiceServer
	backendUseCase *biz.IsBackendAvaliableUsecase
}

func NewHealthService(backendUsecase *biz.IsBackendAvaliableUsecase) *HealthService {
	return &HealthService{
		backendUseCase: backendUsecase,
	}
}

func (s *HealthService) GetLivez(ctx context.Context, req *pb.GetLivezRequest) (*pb.GetLivezResponse, error) {
	return &pb.GetLivezResponse{Status: "OK", Code: 200}, nil
}

func (s *HealthService) GetReadyz(ctx context.Context, req *pb.GetReadyzRequest) (*pb.GetReadyzResponse, error) {
	err := s.backendUseCase.IsBackendAvailable()
	if err != nil {
		return &pb.GetReadyzResponse{Status: "Unavailable", Code: 503}, nil
	}
	return &pb.GetReadyzResponse{Status: "OK", Code: 200}, nil
}
