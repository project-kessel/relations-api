package service

import (
	"context"
	"fmt"

	"github.com/project-kessel/relations-api/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/project-kessel/relations-api/api/relations/v0"
)

type CheckService struct {
	pb.UnimplementedKesselCheckServiceServer
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
	if err := req.ValidateAll(); err != nil {
		s.log.Infof("Request failed to pass validation: %v", req)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	if err := req.Subject.ValidateAll(); err != nil {
		s.log.Infof("Subject failed to pass validation: %v", req)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	if err := req.Resource.ValidateAll(); err != nil {
		s.log.Infof("Resource failed to pass validation: %v", req)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	resp, err := s.check.Check(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to perform check: %w", err)
	}
	return resp, nil
}
