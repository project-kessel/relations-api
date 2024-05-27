package service

import (
	"ciam-rebac/internal/biz"
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "ciam-rebac/api/relations/v0"
)

type RelationshipsService struct {
	pb.UnimplementedKesselTupleServiceServer
	createUsecase *biz.CreateRelationshipsUsecase
	readUsecase   *biz.ReadRelationshipsUsecase
	deleteUsecase *biz.DeleteRelationshipsUsecase
	log           *log.Helper
}

func NewRelationshipsService(logger log.Logger, createUseCase *biz.CreateRelationshipsUsecase, readUsecase *biz.ReadRelationshipsUsecase, deleteUsecase *biz.DeleteRelationshipsUsecase) *RelationshipsService {
	return &RelationshipsService{
		log:           log.NewHelper(logger),
		createUsecase: createUseCase,
		readUsecase:   readUsecase,
		deleteUsecase: deleteUsecase,
	}
}

func (s *RelationshipsService) CreateRelationships(ctx context.Context, req *pb.CreateTuplesRequest) (*pb.CreateTuplesResponse, error) {
	s.log.Infof("Create relationships request: %v", req)

	err := s.createUsecase.CreateRelationships(ctx, req.Tuples, req.GetUpsert()) //The generated .GetUpsert() defaults to false
	if err != nil {
		return nil, err
	}

	return &pb.CreateTuplesResponse{}, nil
}

func (s *RelationshipsService) ReadRelationships(req *pb.ReadTuplesRequest, conn pb.KesselTupleService_ReadTuplesServer) error {
	ctx := conn.Context() //Doesn't get context from grpc?

	relationships, errs, err := s.readUsecase.ReadRelationships(ctx, req)

	if err != nil {
		return err
	}

	for rel := range relationships {
		err = conn.Send(&pb.ReadTuplesResponse{
			Tuple:             rel.Relationship,
			ContinuationToken: string(rel.Continuation),
		})
		if err != nil {
			return err
		}
	}

	err, ok := <-errs
	if ok {
		return err
	}

	return nil
}

func (s *RelationshipsService) DeleteRelationships(ctx context.Context, req *pb.DeleteTuplesRequest) (*pb.DeleteTuplesResponse, error) {
	s.log.Infof("Delete relationships request: %v", req)

	err := s.deleteUsecase.DeleteRelationships(ctx, req.Filter)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteTuplesResponse{}, nil
}
