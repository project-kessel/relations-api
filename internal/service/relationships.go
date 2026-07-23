package service

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/project-kessel/relations-api/internal/biz"

	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
)

type RelationshipsService struct {
	pb.UnimplementedKesselTupleServiceServer
	createUsecase      *biz.CreateRelationshipsUsecase
	readUsecase        *biz.ReadRelationshipsUsecase
	deleteUsecase      *biz.DeleteRelationshipsUsecase
	importBulkUsecase  *biz.ImportBulkTuplesUsecase
	acquireLockUsecase *biz.AcquireLockUsecase
	log                *log.Helper
}

func NewRelationshipsService(logger log.Logger, createUseCase *biz.CreateRelationshipsUsecase, readUsecase *biz.ReadRelationshipsUsecase, deleteUsecase *biz.DeleteRelationshipsUsecase, importBulkUsecase *biz.ImportBulkTuplesUsecase, acquireLockUsecase *biz.AcquireLockUsecase) *RelationshipsService {
	return &RelationshipsService{
		log:                log.NewHelper(logger),
		createUsecase:      createUseCase,
		readUsecase:        readUsecase,
		deleteUsecase:      deleteUsecase,
		importBulkUsecase:  importBulkUsecase,
		acquireLockUsecase: acquireLockUsecase,
	}
}

func (s *RelationshipsService) CreateTuples(ctx context.Context, req *pb.CreateTuplesRequest) (*pb.CreateTuplesResponse, error) {
	resp, err := s.createUsecase.CreateRelationships(ctx, req.Tuples, req.GetUpsert(), req.GetFencingCheck()) //The generated .GetUpsert() defaults to false
	if err != nil {
		// Tuple creation failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation, EOI-11 warnings_or_errors)
		s.log.WithContext(ctx).Warnw(
			"msg", "Tuple creation failed",
			"action", "CREATE",
			"resource_type", "relationship_tuple",
			"resource_id", fmt.Sprintf("count:%d", len(req.Tuples)),
			"outcome", "failure",
			"principal", extractPrincipal(ctx),
			"reason", "spicedb_error",
		)
		return nil, fmt.Errorf("error creating tuples: %w", err)
	}

	// Tuple creation - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation)
	s.log.WithContext(ctx).Infow(
		"msg", "Tuples created",
		"action", "CREATE",
		"resource_type", "relationship_tuple",
		"resource_id", fmt.Sprintf("count:%d", len(req.Tuples)),
		"outcome", "success",
		"principal", extractPrincipal(ctx),
	)

	return &pb.CreateTuplesResponse{ConsistencyToken: resp.GetConsistencyToken()}, nil
}

func (s *RelationshipsService) ReadTuples(req *pb.ReadTuplesRequest, conn pb.KesselTupleService_ReadTuplesServer) error {
	ctx := conn.Context()

	relationships, errs, err := s.readUsecase.ReadRelationships(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving tuples: %w", err)
	}

	for rel := range relationships {
		err = conn.Send(&pb.ReadTuplesResponse{
			Tuple:            rel.Relationship,
			Pagination:       &pb.ResponsePagination{ContinuationToken: string(rel.Continuation)},
			ConsistencyToken: rel.ConsistencyToken,
		})
		if err != nil {
			return fmt.Errorf("error sending retrieved tuple to the client: %w", err)
		}
	}

	err, ok := <-errs
	if ok {
		return fmt.Errorf("error received from Zanzibar backend while streaming tuples: %w", err)
	}

	return nil
}

func (s *RelationshipsService) DeleteTuples(ctx context.Context, req *pb.DeleteTuplesRequest) (*pb.DeleteTuplesResponse, error) {
	resourceID := deleteFilterResourceID(req.Filter)

	resp, err := s.deleteUsecase.DeleteRelationships(ctx, req.Filter, req.GetFencingCheck())
	if err != nil {
		// Tuple deletion failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation, EOI-11 warnings_or_errors)
		s.log.WithContext(ctx).Warnw(
			"msg", "Tuple deletion failed",
			"action", "DELETE",
			"resource_type", "relationship_tuple",
			"resource_id", resourceID,
			"outcome", "failure",
			"principal", extractPrincipal(ctx),
			"reason", "spicedb_error",
		)
		return nil, fmt.Errorf("error deleting tuples: %w", err)
	}

	// Tuple deletion - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation)
	s.log.WithContext(ctx).Infow(
		"msg", "Tuples deleted",
		"action", "DELETE",
		"resource_type", "relationship_tuple",
		"resource_id", resourceID,
		"outcome", "success",
		"principal", extractPrincipal(ctx),
	)

	return &pb.DeleteTuplesResponse{ConsistencyToken: resp.GetConsistencyToken()}, nil
}

func deleteFilterResourceID(filter *pb.RelationTupleFilter) string {
	if filter == nil {
		return "filtered"
	}
	ns := filter.GetResourceNamespace()
	rt := filter.GetResourceType()
	rid := filter.GetResourceId()
	if ns != "" && rt != "" && rid != "" {
		return fmt.Sprintf("%s/%s:%s", ns, rt, rid)
	}
	if ns != "" && rt != "" {
		return fmt.Sprintf("%s/%s", ns, rt)
	}
	if rt != "" {
		return rt
	}
	return "filtered"
}

func (s *RelationshipsService) ImportBulkTuples(stream grpc.ClientStreamingServer[pb.ImportBulkTuplesRequest, pb.ImportBulkTuplesResponse]) error {
	ctx := stream.Context()
	err := s.importBulkUsecase.ImportBulkTuples(stream)
	if err != nil {
		// Bulk tuple import failure - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation, EOI-11 warnings_or_errors)
		s.log.WithContext(ctx).Warnw(
			"msg", "Bulk tuple import failed",
			"action", "IMPORT",
			"resource_type", "relationship_tuple",
			"resource_id", "bulk_import",
			"outcome", "failure",
			"principal", extractPrincipal(ctx),
			"reason", "import_error",
		)
		return fmt.Errorf("error import bulk tuples: %w", err)
	}

	// Bulk tuple import - SEC-MON-REQ-1 compliance (EOI-1 pii_manipulation, EOI-4 access_manipulation)
	s.log.WithContext(ctx).Infow(
		"msg", "Bulk tuples imported",
		"action", "IMPORT",
		"resource_type", "relationship_tuple",
		"resource_id", "bulk_import",
		"outcome", "success",
		"principal", extractPrincipal(ctx),
	)
	return nil
}

func (s *RelationshipsService) AcquireLock(ctx context.Context, req *pb.AcquireLockRequest) (*pb.AcquireLockResponse, error) {
	resp, err := s.acquireLockUsecase.AcquireLock(ctx, req)
	if err != nil {
		// Lock acquisition failure - SEC-MON-REQ-1 compliance (EOI-4 access_manipulation, EOI-11 warnings_or_errors)
		s.log.WithContext(ctx).Warnw(
			"msg", "Lock acquisition failed",
			"action", "CREATE",
			"resource_type", "lock",
			"resource_id", req.GetLockId(),
			"outcome", "failure",
			"principal", extractPrincipal(ctx),
			"reason", "lock_error",
		)
		return nil, fmt.Errorf("error acquiring lock: %w", err)
	}

	// Lock acquisition - SEC-MON-REQ-1 compliance (EOI-4 access_manipulation)
	s.log.WithContext(ctx).Infow(
		"msg", "Lock acquired",
		"action", "CREATE",
		"resource_type", "lock",
		"resource_id", req.GetLockId(),
		"outcome", "success",
		"principal", extractPrincipal(ctx),
	)
	return resp, nil
}
