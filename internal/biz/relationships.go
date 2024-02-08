package biz

import (
	v1 "ciam-rebac/api/rebac/v1"
	"context"
	"github.com/go-kratos/kratos/v2/log"
)

// relationship domain objects re-used from the api layer for now, but otherwise would be defined here
type TouchSemantics bool

type ZanzibarRepository interface {
	CreateRelationships(context.Context, []*v1.Relationship, TouchSemantics) error
	ReadRelationships(context.Context, []*v1.RelationshipFilter) ([]*v1.Relationship, error)
	DeleteRelationships(context.Context, []*v1.RelationshipFilter) ([]*v1.Relationship, error)
}

type CreateRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCreateRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *CreateRelationshipsUsecase {
	return &CreateRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CreateRelationshipsUsecase) CreateRelationships(ctx context.Context, r []*v1.Relationship, touch bool) error {
	rc.log.WithContext(ctx).Infof("CreateRelationships: %v %s", r, touch)
	return rc.repo.CreateRelationships(ctx, r, TouchSemantics(touch))
}
