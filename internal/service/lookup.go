package service

import (
	pb "ciam-rebac/api/relations/v0"
	"ciam-rebac/internal/biz"
	"context"
)

type LookupService struct {
	pb.UnimplementedLookupServer
	repo biz.ZanzibarRepository
}

func NewLookupSubjectsService(repo biz.ZanzibarRepository) *LookupService {
	return &LookupService{
		repo: repo,
	}

}

func (s *LookupService) Subjects(req *pb.LookupSubjectsRequest, conn pb.Lookup_SubjectsServer) error {
	ctx := context.TODO() //Doesn't get context from grpc?
	limit := uint32(1000)
	if req.Limit != nil {
		limit = *req.Limit
	}

	continuation := biz.ContinuationToken("")
	if req.ContinuationToken != nil {
		continuation = biz.ContinuationToken(*req.ContinuationToken)
	}
	subs, errs, err := s.repo.LookupSubjects(ctx, req.SubjectType, req.Relation, &pb.ObjectReference{
		Type: req.Object.Type, //Need null check
		Id:   req.Object.Id,
	}, limit, continuation)

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
