package service

import (
	"ciam-rebac/internal/biz"
	"context"

	"github.com/go-kratos/kratos/v2/log"

	pb "ciam-rebac/api/rebac/v1"
)

type RelationshipsService struct {
	pb.UnimplementedRelationshipsServer
	createUsecase *biz.CreateRelationshipsUsecase
	log           *log.Helper
}

func NewRelationshipsService(logger log.Logger, createUseCase *biz.CreateRelationshipsUsecase) *RelationshipsService {
	return &RelationshipsService{log: log.NewHelper(logger), createUsecase: createUseCase}
}

func (s *RelationshipsService) CreateRelationships(ctx context.Context, req *pb.CreateRelationshipsRequest) (*pb.CreateRelationshipsResponse, error) {
	s.log.Infof("Create relationships request: %v", req)

	s.createUsecase.CreateRelationships(ctx, req.Relationships, req.GetTouch())

	return &pb.CreateRelationshipsResponse{}, nil
}
func (s *RelationshipsService) ReadRelationships(ctx context.Context, req *pb.ReadRelationshipsRequest) (*pb.ReadRelationshipsResponse, error) {
	s.log.Infof("Read relationships request: %v", req)
	return &pb.ReadRelationshipsResponse{}, nil
}
func (s *RelationshipsService) DeleteRelationships(ctx context.Context, req *pb.DeleteRelationshipsRequest) (*pb.DeleteRelationshipsResponse, error) {
	s.log.Infof("Delete relationships request: %v", req)
	return &pb.DeleteRelationshipsResponse{}, nil
}
