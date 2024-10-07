package biz

import (
	"context"
	"google.golang.org/grpc"

	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"

	"github.com/go-kratos/kratos/v2/log"
)

// relationship domain objects re-used from the api layer for now, but otherwise would be defined here
type TouchSemantics bool

type ContinuationToken string
type SubjectResult struct {
	Subject      *v1beta1.SubjectReference
	Continuation ContinuationToken
}
type ResourceResult struct {
	Resource     *v1beta1.ObjectReference
	Continuation ContinuationToken
}

type RelationshipResult struct {
	Relationship *v1beta1.Relationship
	Continuation ContinuationToken
}

type ZanzibarRepository interface {
	Check(ctx context.Context, request *v1beta1.CheckRequest) (*v1beta1.CheckResponse, error)
	CreateRelationships(context.Context, []*v1beta1.Relationship, TouchSemantics) error
	ReadRelationships(ctx context.Context, filter *v1beta1.RelationTupleFilter, limit uint32, continuation ContinuationToken) (chan *RelationshipResult, chan error, error)
	DeleteRelationships(context.Context, *v1beta1.RelationTupleFilter) error
	LookupSubjects(ctx context.Context, subjectType *v1beta1.ObjectType, subject_relation, relation string, resource *v1beta1.ObjectReference, limit uint32, continuation ContinuationToken) (chan *SubjectResult, chan error, error)
	LookupResources(ctx context.Context, resouce_type *v1beta1.ObjectType, relation string, subject *v1beta1.SubjectReference, limit uint32, continuation ContinuationToken) (chan *ResourceResult, chan error, error)
	IsBackendAvailable() error
	ImportBulkTuples(stream grpc.ClientStreamingServer[v1beta1.ImportBulkTuplesRequest, v1beta1.ImportBulkTuplesResponse]) error
}

type CheckUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCheckUsecase(repo ZanzibarRepository, logger log.Logger) *CheckUsecase {
	return &CheckUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CheckUsecase) Check(ctx context.Context, check *v1beta1.CheckRequest) (*v1beta1.CheckResponse, error) {
	return rc.repo.Check(ctx, check)
}

type CreateRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCreateRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *CreateRelationshipsUsecase {
	return &CreateRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CreateRelationshipsUsecase) CreateRelationships(ctx context.Context, r []*v1beta1.Relationship, touch bool) error {
	return rc.repo.CreateRelationships(ctx, r, TouchSemantics(touch))
}

type ReadRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewReadRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *ReadRelationshipsUsecase {
	return &ReadRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *ReadRelationshipsUsecase) ReadRelationships(ctx context.Context, req *v1beta1.ReadTuplesRequest) (chan *RelationshipResult, chan error, error) {
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

func (rc *DeleteRelationshipsUsecase) DeleteRelationships(ctx context.Context, r *v1beta1.RelationTupleFilter) error {
	return rc.repo.DeleteRelationships(ctx, r)
}

type ImportBulkTuplesUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewImportBulkTuplesUsecase(repo ZanzibarRepository, logger log.Logger) *ImportBulkTuplesUsecase {
	return &ImportBulkTuplesUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *ImportBulkTuplesUsecase) ImportBulkTuples(client grpc.ClientStreamingServer[v1beta1.ImportBulkTuplesRequest, v1beta1.ImportBulkTuplesResponse]) error {
	return rc.repo.ImportBulkTuples(client)
}
