package service

import (
	"ciam-rebac/internal/biz"
	"context"
	"github.com/go-kratos/kratos/v2/log"

	pb "ciam-rebac/api/rebac/v1"
)

type CheckService struct {
	pb.UnimplementedCheckServer
	check *biz.CheckUsecase
	log   *log.Helper
}

func NewCheckService(logger log.Logger, checkUseCase *biz.CheckUsecase) *CheckService {
	return &CheckService{
		check: checkUseCase,
		log:   log.NewHelper(logger),
	}
}

func (s *CheckService) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	s.log.Infof("Check permission: %v", req)
	resp, err := s.check.Check(ctx, req)
	if err != nil {
		s.log.Errorf("Failed to perform check %v", err)
		return resp, err
	}
	return resp, nil
}
