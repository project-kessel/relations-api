package service

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
)

type LookupService struct {
	pb.UnimplementedKesselLookupServiceServer
	subjectsUsecase  *biz.GetSubjectsUsecase
	resourcesUsecase *biz.GetResourcesUsecase
	log              *log.Helper
}

func NewLookupService(logger log.Logger, subjectsUseCase *biz.GetSubjectsUsecase, resourcesUsecase *biz.GetResourcesUsecase) *LookupService {
	return &LookupService{
		subjectsUsecase:  subjectsUseCase,
		resourcesUsecase: resourcesUsecase,
		log:              log.NewHelper(logger),
	}

}

func (s *LookupService) LookupSubjects(req *pb.LookupSubjectsRequest, conn pb.KesselLookupService_LookupSubjectsServer) error {
	ctx := conn.Context()

	subs, errs, err := s.subjectsUsecase.Get(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving subjects: %w", err)
	}

	for sub := range subs {
		err = conn.Send(&pb.LookupSubjectsResponse{
			Subject:          sub.Subject,
			Pagination:       &pb.ResponsePagination{ContinuationToken: string(sub.Continuation)},
			ConsistencyToken: sub.ConsistencyToken,
		})
		if err != nil {
			return fmt.Errorf("error sending retrieved subject to the client: %w", err)
		}
	}

	err, ok := <-errs
	if ok {
		return fmt.Errorf("error received while streaming subjects from Zanzibar backend: %w", err)
	}

	return nil
}

func (s *LookupService) LookupResources(req *pb.LookupResourcesRequest, conn pb.KesselLookupService_LookupResourcesServer) error {
	ctx := conn.Context()

	res, errs, err := s.resourcesUsecase.Get(ctx, req)

	if err != nil {
		return fmt.Errorf("failed to retrieve resources: %w", err)
	}
	for re := range res {
		err = conn.Send(&pb.LookupResourcesResponse{
			Resource:         re.Resource,
			Pagination:       &pb.ResponsePagination{ContinuationToken: string(re.Continuation)},
			ConsistencyToken: re.ConsistencyToken,
		})
		if err != nil {
			return fmt.Errorf("error sending retrieved resource to the client: %w", err)
		}
	}
	err, ok := <-errs
	if ok {
		return fmt.Errorf("error received while streaming subjects from Zanzibar backend: %w", err)
	}

	return nil
}
