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
	createUsecase     *biz.CreateRelationshipsUsecase
	readUsecase       *biz.ReadRelationshipsUsecase
	deleteUsecase     *biz.DeleteRelationshipsUsecase
	importBulkUsecase *biz.ImportBulkTuplesUsecase
	log               *log.Helper
}

func NewRelationshipsService(logger log.Logger, createUseCase *biz.CreateRelationshipsUsecase, readUsecase *biz.ReadRelationshipsUsecase, deleteUsecase *biz.DeleteRelationshipsUsecase, importBulkUsecase *biz.ImportBulkTuplesUsecase) *RelationshipsService {
	return &RelationshipsService{
		log:               log.NewHelper(logger),
		createUsecase:     createUseCase,
		readUsecase:       readUsecase,
		deleteUsecase:     deleteUsecase,
		importBulkUsecase: importBulkUsecase,
	}
}

func (s *RelationshipsService) CreateTuples(ctx context.Context, req *pb.CreateTuplesRequest) (*pb.CreateTuplesResponse, error) {
	resp, err := s.createUsecase.CreateRelationships(ctx, req.Tuples, req.GetUpsert()) //The generated .GetUpsert() defaults to false
	if err != nil {
		return nil, fmt.Errorf("error creating tuples: %w", err)
	}

	return &pb.CreateTuplesResponse{CreatedAt: resp.GetCreatedAt()}, nil
}

func (s *RelationshipsService) ReadTuples(req *pb.ReadTuplesRequest, conn pb.KesselTupleService_ReadTuplesServer) error {
	ctx := conn.Context()

	relationships, errs, err := s.readUsecase.ReadRelationships(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving tuples: %w", err)
	}

	for rel := range relationships {
		err = conn.Send(&pb.ReadTuplesResponse{
			Tuple:      rel.Relationship,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(rel.Continuation)},
			ReadAt:     rel.Zookie,
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
	resp, err := s.deleteUsecase.DeleteRelationships(ctx, req.Filter)
	if err != nil {
		return nil, fmt.Errorf("error deleting tuples: %w", err)
	}

	return &pb.DeleteTuplesResponse{DeletedAt: resp.GetDeletedAt()}, nil
}

func (s *RelationshipsService) ImportBulkTuples(stream grpc.ClientStreamingServer[pb.ImportBulkTuplesRequest, pb.ImportBulkTuplesResponse]) error {
	err := s.importBulkUsecase.ImportBulkTuples(stream)
	if err != nil {
		return fmt.Errorf("error import bulk tuples: %w", err)
	}
	return nil
}
