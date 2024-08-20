package service

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/project-kessel/relations-api/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"

	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
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

func (s *RelationshipsService) CreateTuples(ctx context.Context, req *pb.CreateTuplesRequest) (*pb.CreateTuplesResponse, error) {
	v, err := protovalidate.New()
	if err != nil {
		s.log.Errorf("failed to initialize validator: ", err)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	for idx := range req.Tuples {
		if err = v.Validate(req.Tuples[idx]); err != nil {
			s.log.Infof("Request failed to pass validation: %v", req.Tuples[idx])
			return nil, errors.BadRequest("Invalid request", err.Error())
		}
	}

	err = s.createUsecase.CreateRelationships(ctx, req.Tuples, req.GetUpsert()) //The generated .GetUpsert() defaults to false
	if err != nil {
		return nil, fmt.Errorf("error creating tuples: %w", err)
	}

	return &pb.CreateTuplesResponse{}, nil
}

func (s *RelationshipsService) ReadTuples(req *pb.ReadTuplesRequest, conn pb.KesselTupleService_ReadTuplesServer) error {
	v, err := protovalidate.New()
	if err != nil {
		s.log.Errorf("failed to initialize validator: ", err)
		return errors.BadRequest("Invalid request", err.Error())
	}

	if err = v.Validate(req); err != nil {
		s.log.Infof("Request failed to pass validation: %v", req)
		return errors.BadRequest("Invalid request", err.Error())
	}

	ctx := conn.Context()

	relationships, errs, err := s.readUsecase.ReadRelationships(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving tuples: %w", err)
	}

	for rel := range relationships {
		err = conn.Send(&pb.ReadTuplesResponse{
			Tuple:      rel.Relationship,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(rel.Continuation)},
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
	v, err := protovalidate.New()
	if err != nil {
		s.log.Errorf("failed to initialize validator: ", err)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	if err = v.Validate(req); err != nil {
		s.log.Infof("Request failed to pass validation: %v", req)
		return nil, errors.BadRequest("Invalid request", err.Error())
	}

	err = s.deleteUsecase.DeleteRelationships(ctx, req.Filter)
	if err != nil {
		return nil, fmt.Errorf("error deleting tuples: %w", err)
	}

	return &pb.DeleteTuplesResponse{}, nil
}
