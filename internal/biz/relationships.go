package biz

import (
	"context"

	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"

	"github.com/go-kratos/kratos/v2/log"
)

// relationship domain objects re-used from the api layer for now, but otherwise would be defined here
type TouchSemantics bool

type ContinuationToken string
type SubjectResult struct {
	Subject          *v1beta1.SubjectReference
	Continuation     ContinuationToken
	ConsistencyToken *v1beta1.ConsistencyToken
}
type ResourceResult struct {
	Resource         *v1beta1.ObjectReference
	Continuation     ContinuationToken
	ConsistencyToken *v1beta1.ConsistencyToken
}

type RelationshipResult struct {
	Relationship     *v1beta1.Relationship
	Continuation     ContinuationToken
	ConsistencyToken *v1beta1.ConsistencyToken
}

type ZanzibarRepository interface {
	Check(ctx context.Context, request *v1beta1.CheckRequest) (*v1beta1.CheckResponse, error)
	CheckForUpdate(ctx context.Context, request *v1beta1.CheckForUpdateRequest) (*v1beta1.CheckForUpdateResponse, error)
	CheckBulk(ctx context.Context, request *v1beta1.CheckBulkRequest) (*v1beta1.CheckBulkResponse, error)
	CreateRelationships(context.Context, []*v1beta1.Relationship, TouchSemantics, *v1beta1.FencingCheck) (*v1beta1.CreateTuplesResponse, error)
	ReadRelationships(ctx context.Context, filter *v1beta1.RelationTupleFilter, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *RelationshipResult, chan error, error)
	DeleteRelationships(context.Context, *v1beta1.RelationTupleFilter, *v1beta1.FencingCheck) (*v1beta1.DeleteTuplesResponse, error)
	LookupSubjects(ctx context.Context, subjectType *v1beta1.ObjectType, subject_relation, relation string, resource *v1beta1.ObjectReference, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *SubjectResult, chan error, error)
	LookupResources(ctx context.Context, resouce_type *v1beta1.ObjectType, relation string, subject *v1beta1.SubjectReference, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *ResourceResult, chan error, error)
	IsBackendAvailable() error
	ImportBulkTuples(stream grpc.ClientStreamingServer[v1beta1.ImportBulkTuplesRequest, v1beta1.ImportBulkTuplesResponse]) error
	AcquireLock(ctx context.Context, lockId string) (*v1beta1.AcquireLockResponse, error)
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

type CheckForUpdateUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

type CheckBulkUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCheckBulkUsecase(repo ZanzibarRepository, logger log.Logger) *CheckBulkUsecase {
	return &CheckBulkUsecase{repo: repo, log: log.NewHelper(logger)}
}

func NewCheckForUpdateUsecase(repo ZanzibarRepository, logger log.Logger) *CheckForUpdateUsecase {
	return &CheckForUpdateUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CheckForUpdateUsecase) CheckForUpdate(ctx context.Context, check *v1beta1.CheckForUpdateRequest) (*v1beta1.CheckForUpdateResponse, error) {
	return rc.repo.CheckForUpdate(ctx, check)
}

func (rc *CheckBulkUsecase) CheckBulk(ctx context.Context, check *v1beta1.CheckBulkRequest) (*v1beta1.CheckBulkResponse, error) {
	return rc.repo.CheckBulk(ctx, check)
}

// CheckBulkForUpdateUsecase runs N strongly-consistent CheckForUpdate checks (one per item) and returns pairs in order.
type CheckBulkForUpdateUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

// NewCheckBulkForUpdateUsecase creates a usecase that delegates each item to repo.CheckForUpdate.
func NewCheckBulkForUpdateUsecase(repo ZanzibarRepository, logger log.Logger) *CheckBulkForUpdateUsecase {
	return &CheckBulkForUpdateUsecase{repo: repo, log: log.NewHelper(logger)}
}

// CheckBulkForUpdate runs one CheckForUpdate per request item and returns one pair per item in the same order.
func (u *CheckBulkForUpdateUsecase) CheckBulkForUpdate(ctx context.Context, req *v1beta1.CheckBulkForUpdateRequest) (*v1beta1.CheckBulkForUpdateResponse, error) {
	pairs := make([]*v1beta1.CheckBulkResponsePair, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		cfuReq := &v1beta1.CheckForUpdateRequest{
			Resource: item.GetResource(),
			Relation: item.GetRelation(),
			Subject:  item.GetSubject(),
		}
		resp, err := u.repo.CheckForUpdate(ctx, cfuReq)
		if err != nil {
			pairs = append(pairs, pairWithError(item, err))
			continue
		}
		allowed := v1beta1.CheckBulkResponseItem_ALLOWED_FALSE
		if resp.GetAllowed() == v1beta1.CheckForUpdateResponse_ALLOWED_TRUE {
			allowed = v1beta1.CheckBulkResponseItem_ALLOWED_TRUE
		}
		pairs = append(pairs, &v1beta1.CheckBulkResponsePair{
			Request: item,
			Response: &v1beta1.CheckBulkResponsePair_Item{
				Item: &v1beta1.CheckBulkResponseItem{Allowed: allowed},
			},
		})
	}
	return &v1beta1.CheckBulkForUpdateResponse{Pairs: pairs}, nil
}

// pairWithError builds a CheckBulkResponsePair with the request item and a gRPC Status error.
func pairWithError(item *v1beta1.CheckBulkRequestItem, err error) *v1beta1.CheckBulkResponsePair {
	s := status.Convert(err)
	return &v1beta1.CheckBulkResponsePair{
		Request: item,
		Response: &v1beta1.CheckBulkResponsePair_Error{
			Error: &rpcstatus.Status{Code: int32(s.Code()), Message: s.Message()},
		},
	}
}

type CreateRelationshipsUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewCreateRelationshipsUsecase(repo ZanzibarRepository, logger log.Logger) *CreateRelationshipsUsecase {
	return &CreateRelationshipsUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *CreateRelationshipsUsecase) CreateRelationships(ctx context.Context, r []*v1beta1.Relationship, touch bool, fencing *v1beta1.FencingCheck) (*v1beta1.CreateTuplesResponse, error) {
	return rc.repo.CreateRelationships(ctx, r, TouchSemantics(touch), fencing)
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

	relationships, errs, err := rc.repo.ReadRelationships(ctx, req.Filter, limit, continuation, req.GetConsistency())

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

func (rc *DeleteRelationshipsUsecase) DeleteRelationships(ctx context.Context, r *v1beta1.RelationTupleFilter, fencing *v1beta1.FencingCheck) (*v1beta1.DeleteTuplesResponse, error) {
	return rc.repo.DeleteRelationships(ctx, r, fencing)
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

type AcquireLockUsecase struct {
	repo ZanzibarRepository
	log  *log.Helper
}

func NewAcquireLockUsecase(repo ZanzibarRepository, logger log.Logger) *AcquireLockUsecase {
	return &AcquireLockUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (rc *AcquireLockUsecase) AcquireLock(ctx context.Context, req *v1beta1.AcquireLockRequest) (*v1beta1.AcquireLockResponse, error) {
	return rc.repo.AcquireLock(ctx, req.LockId)
}
