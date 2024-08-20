package service

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/project-kessel/relations-api/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
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
	v, err := protovalidate.New()
	if err != nil {
		s.log.Errorf("failed to initialize validator: ", err)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	if err = v.Validate(req); err != nil {
		s.log.Infof("Request failed to pass validation: %v", req)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	resp, err := s.check.Check(ctx, req)
	if err != nil {
		return resp, fmt.Errorf("failed to perform check: %w", err)
	}
	return resp, nil
}
