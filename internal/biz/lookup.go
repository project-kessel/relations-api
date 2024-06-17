package biz

import (
	"context"
	v0 "github.com/project-kessel/relations-api/api/relations/v0"

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

	if err := req.Validate(); err != nil {
		s.log.WithContext(ctx).Error("Request failed to pass validation: %v", req)
		return nil, nil, errors.BadRequest("Invalid request", err.Error())
	}

	if err := req.Resource.Validate(); err != nil {
		s.log.WithContext(ctx).Error("Request failed to pass validation: %v", req)
		return nil, nil, errors.BadRequest("Invalid request", err.Error())
	}

	if req.SubjectRelation != nil {
		subjectRelation = *req.SubjectRelation
	}

	subs, errs, err := s.repo.LookupSubjects(ctx, req.SubjectType, subjectRelation, req.Relation, &v0.ObjectReference{
		Type: req.Resource.Type,
		Id:   req.Resource.Id,
	}, limit, continuation)

	if err != nil {
		return nil, nil, err
	}

	return subs, errs, nil
}
