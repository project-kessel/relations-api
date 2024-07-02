package biz

import (
	"context"

	v0 "github.com/project-kessel/relations-api/api/kessel/relations/v0"

	"github.com/go-kratos/kratos/v2/log"
)

// relationship domain objects re-used from the api layer for now, but otherwise would be defined here
type TouchSemantics bool

type ContinuationToken string
type SubjectResult struct {
	Subject      *v0.SubjectReference
	Continuation ContinuationToken
}
type ResourceResult struct {
	Resource     *v0.ObjectReference
	Continuation ContinuationToken
}

type RelationshipResult struct {
	Relationship *v0.Relationship
	Continuation ContinuationToken
}

type ZanzibarRepository interface {
	Check(ctx context.Context, request *v0.CheckRequest) (*v0.CheckResponse, error)
	CreateRelationships(context.Context, []*v0.Relationship, TouchSemantics) error
	ReadRelationships(ctx context.Context, filter *v0.RelationTupleFilter, limit uint32, continuation ContinuationToken) (chan *RelationshipResult, chan error, error)
	DeleteRelationships(context.Context, *v0.RelationTupleFilter) error
	LookupSubjects(ctx context.Context, subjectType *v0.ObjectType, subject_relation, relation string, resource *v0.ObjectReference, limit uint32, continuation ContinuationToken) (chan *SubjectResult, chan error, error)
	LookupResources(ctx context.Context, resouce_type *v0.ObjectType, relation string, subject *v0.SubjectReference, limit uint32, continuation ContinuationToken) (chan *ResourceResult, chan error, error)
	IsBackendAvailable() error
}

type CheckUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCheckUsecase(repo ZanzibarRepository, logger log.Logger) *CheckUsecase {
	return &CheckUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CheckUsecase) Check(ctx context.Context, check *v0.CheckRequest) (*v0.CheckResponse, error) {
	return rc.repo.Check(ctx, check)
}

type CreateRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCreateRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *CreateRelationshipsUsecase {
	return &CreateRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CreateRelationshipsUsecase) CreateRelationships(ctx context.Context, r []*v0.Relationship, touch bool) error {
	return rc.repo.CreateRelationships(ctx, r, TouchSemantics(touch))
}

type ReadRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewReadRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *ReadRelationshipsUsecase {
	return &ReadRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *ReadRelationshipsUsecase) ReadRelationships(ctx context.Context, req *v0.ReadTuplesRequest) (chan *RelationshipResult, chan error, error) {
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

	relationships, errs, err := rc.repo.ReadRelationships(ctx, req.Filter, limit, continuation)

	if err != nil {
		return nil, nil, err
	}

	return relationships, errs, nil
}

type DeleteRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewDeleteRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *DeleteRelationshipsUsecase {
	return &DeleteRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *DeleteRelationshipsUsecase) DeleteRelationships(ctx context.Context, r *v0.RelationTupleFilter) error {
	return rc.repo.DeleteRelationships(ctx, r)
}
