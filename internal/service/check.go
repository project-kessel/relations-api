package service

import (
	"context"
	"fmt"

	"github.com/project-kessel/relations-api/internal/biz"

	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
)

type CheckService struct {
	pb.UnimplementedKesselCheckServiceServer
	check          *biz.CheckUsecase
	checkForUpdate *biz.CheckForUpdateUsecase
	log            *log.Helper
}

func NewCheckService(logger log.Logger, checkUseCase *biz.CheckUsecase, checkForUpdateUseCase *biz.CheckForUpdateUsecase) *CheckService {
	return &CheckService{
		check:          checkUseCase,
		checkForUpdate: checkForUpdateUseCase,
		log:            log.NewHelper(logger),
	}
}

func (s *CheckService) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	resp, err := s.check.Check(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to perform check: %w", err)
	}
	return resp, nil
}

func (s *CheckService) CheckForUpdate(ctx context.Context, req *pb.CheckForUpdateRequest) (*pb.CheckForUpdateResponse, error) {
	resp, err := s.checkForUpdate.CheckForUpdate(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to perform checkForUpdate: %w", err)
	}
	return resp, nil
}
