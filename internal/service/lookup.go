package service

import (
	pb "ciam-rebac/api/relations/v0"
	"ciam-rebac/internal/biz"
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
	ctx := conn.Context()

	subs, errs, err := s.subjectsUsecase.Get(ctx, req)

	if err != nil {
		return err
	}

	for sub := range subs {
		err = conn.Send(&pb.LookupSubjectsResponse{
			Subject:           sub.Subject,
			ContinuationToken: string(sub.Continuation),
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
