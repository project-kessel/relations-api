package biz

import (
	v0 "ciam-rebac/api/relations/v0"
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

// relationship domain objects re-used from the api layer for now, but otherwise would be defined here
type TouchSemantics bool

type ZanzibarRepository interface {
	Check(ctx context.Context, request *v0.CheckRequest) (*v0.CheckResponse, error)
	CreateRelationships(context.Context, []*v0.Relationship, TouchSemantics) error
	ReadRelationships(context.Context, *v0.RelationTupleFilter) ([]*v0.Relationship, error)
	DeleteRelationships(context.Context, *v0.RelationTupleFilter) error
}

type CheckUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCheckUsecase(repo ZanzibarRepository, logger log.Logger) *CheckUsecase {
	return &CheckUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CheckUsecase) Check(ctx context.Context, check *v0.CheckRequest) (*v0.CheckResponse, error) {
	rc.log.WithContext(ctx).Infof("Check: %v", check)
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
	rc.log.WithContext(ctx).Infof("CreateRelationships: %v %s", r, touch)
	return rc.repo.CreateRelationships(ctx, r, TouchSemantics(touch))
}

type ReadRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewReadRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *ReadRelationshipsUsecase {
	return &ReadRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *ReadRelationshipsUsecase) ReadRelationships(ctx context.Context, r *v0.RelationTupleFilter) ([]*v0.Relationship, error) {
	rc.log.WithContext(ctx).Infof("ReadRelationships: %v", r)
	return rc.repo.ReadRelationships(ctx, r)
}

type DeleteRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewDeleteRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *DeleteRelationshipsUsecase {
	return &DeleteRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *DeleteRelationshipsUsecase) DeleteRelationships(ctx context.Context, r *v0.RelationTupleFilter) error {
	rc.log.WithContext(ctx).Infof("DeleteRelationships: %v", r)
	return rc.repo.DeleteRelationships(ctx, r)
}
