package service

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/project-kessel/relations-api/api/relations/v0"
	"github.com/project-kessel/relations-api/internal/biz"
)

type LookupService struct {
	pb.UnimplementedKesselLookupServiceServer
	subjectsUsecase *biz.GetSubjectsUsecase
	log             *log.Helper
}

func NewLookupService(logger log.Logger, subjectsUseCase *biz.GetSubjectsUsecase) *LookupService {
	return &LookupService{
		subjectsUsecase: subjectsUseCase,
		log:             log.NewHelper(logger),
	}

}

func (s *LookupService) LookupSubjects(req *pb.LookupSubjectsRequest, conn pb.KesselLookupService_LookupSubjectsServer) error {
	if err := req.ValidateAll(); err != nil {
		s.log.Infof("Request failed to pass validation: %v", req)
		return errors.BadRequest("Invalid request", err.Error())
	}

	if err := req.Resource.ValidateAll(); err != nil {
		s.log.Infof("Resource failed to pass validation: %v", req)
		return errors.BadRequest("Invalid request", err.Error())
	}

	ctx := conn.Context()

	subs, errs, err := s.subjectsUsecase.Get(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving subjects (original request: %v): %w", req, err)
	}

	for sub := range subs {
		err = conn.Send(&pb.LookupSubjectsResponse{
			Subject:    sub.Subject,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(sub.Continuation)},
		})
		if err != nil {
			return fmt.Errorf("error sending retrieved subject to the client (original request: %v): %w", req, err)
		}
	}

	err, ok := <-errs
	if ok {
		return fmt.Errorf("error received while streaming subjects from Zanzibar backend (original request: %v) %w", req, err)
	}

	return nil
}
