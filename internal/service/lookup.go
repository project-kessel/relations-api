package service

import (
	"github.com/go-kratos/kratos/v2/errors"
	pb "github.com/project-kessel/relations-api/api/relations/v0"
	"github.com/project-kessel/relations-api/internal/biz"
)

type LookupService struct {
	pb.UnimplementedKesselLookupServiceServer
	subjectsUsecase *biz.GetSubjectsUsecase
}

func NewLookupService(subjectsUseCase *biz.GetSubjectsUsecase) *LookupService {
	return &LookupService{
		subjectsUsecase: subjectsUseCase,
	}

}

func (s *LookupService) LookupSubjects(req *pb.LookupSubjectsRequest, conn pb.KesselLookupService_LookupSubjectsServer) error {
	if err := req.ValidateAll(); err != nil {
		return errors.BadRequest("Invalid request", err.Error())
	}

	if err := req.Resource.ValidateAll(); err != nil {
		return errors.BadRequest("Invalid request", err.Error())
	}

	ctx := conn.Context()

	subs, errs, err := s.subjectsUsecase.Get(ctx, req)

	if err != nil {
		return err
	}

	for sub := range subs {
		err = conn.Send(&pb.LookupSubjectsResponse{
			Subject:    sub.Subject,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(sub.Continuation)},
		})
		if err != nil {
			return err
		}
	}

	err, ok := <-errs
	if ok {
		return err
	}

	return nil
}
