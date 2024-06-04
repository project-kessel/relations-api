package biz

import (
	v0 "ciam-rebac/api/relations/v0"
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	MaxStreamingCount uint32 = 1000
)

type GetSubjectsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewGetSubjectsUseCase(repo ZanzibarRepository, logger log.Logger) *GetSubjectsUsecase {
	return &GetSubjectsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (s *GetSubjectsUsecase) Get(ctx context.Context, req *v0.LookupSubjectsRequest) (chan *SubjectResult, chan error, error) {
	limit := uint32(MaxStreamingCount)
	continuation := ContinuationToken("")
	subjectRelation := ""

	if req.Pagination != nil {
		if req.Pagination.Limit < limit {
			limit = req.Pagination.Limit
		}

		if req.Pagination.ContinuationToken != nil {
			continuation = ContinuationToken(*req.Pagination.ContinuationToken)
		}
	}

	if req.Resource == nil {
		return nil, nil, errors.BadRequest("Invalid request", "Object is required")
	}

	if req.SubjectRelation != nil {
		subjectRelation = *req.SubjectRelation
	}

	subs, errs, err := s.repo.LookupSubjects(ctx, req.SubjectType, subjectRelation, req.Relation, &v0.ObjectReference{
		Type: req.Resource.Type, //Need null check
		Id:   req.Resource.Id,
	}, limit, continuation)

	if err != nil {
		return nil, nil, err
	}

	return subs, errs, nil
}
