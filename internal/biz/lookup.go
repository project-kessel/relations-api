package biz

import (
	"context"

	v0 "github.com/project-kessel/relations-api/api/kessel/relations/v0"
)

const (
	MaxStreamingCount uint32 = 1000
)

type GetSubjectsUsecase struct {
	repo ZanzibarRepository
}

type GetResourcesUsecase struct {
	repo ZanzibarRepository
}

func NewGetResourcesUseCase(repo ZanzibarRepository) *GetResourcesUsecase {
	return &GetResourcesUsecase{
		repo: repo,
	}
}

func NewGetSubjectsUseCase(repo ZanzibarRepository) *GetSubjectsUsecase {
	return &GetSubjectsUsecase{repo: repo}
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

func (r *GetResourcesUsecase) Get(ctx context.Context, req *v0.LookupResourcesRequest) (chan *ResourceResult, chan error, error) {
	limit := uint32(MaxStreamingCount)
	continuation := ContinuationToken("")

	if req.Pagination != nil {
		if req.Pagination.Limit < limit {
			limit = req.Pagination.Limit
		}

		if req.Pagination.ContinuationToken != nil {
			continuation = ContinuationToken(*req.Pagination.ContinuationToken)
		}
	}
	resources, errs, err := r.repo.LookupResources(ctx, req.ResourceType, req.Relation, req.Subject, limit, continuation)
	if err != nil {
		return nil, nil, err
	}
	return resources, errs, nil
}
