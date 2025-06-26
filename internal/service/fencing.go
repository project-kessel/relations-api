package service

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
)

type FencingService struct {
	pb.UnimplementedKesselFencingServiceServer
	acquireLockUsecase *biz.AcquireLockUsecase
	log                *log.Helper
}

func NewFencingService(logger log.Logger, acquireLockUsecase *biz.AcquireLockUsecase) *FencingService {
	return &FencingService{
		log:                log.NewHelper(logger),
		acquireLockUsecase: acquireLockUsecase,
	}
}

func (s *FencingService) AcquireLock(ctx context.Context, req *pb.AcquireLockRequest) (*pb.AcquireLockResponse, error) {
	newLock, err := s.acquireLockUsecase.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error acquiring lock: %w", err)
	}

	return newLock, nil
}
