package service

import (
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/go-kratos/kratos/v2/errors"
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

	subs, errs, err := s.subjectsUsecase.Get(ctx, req)

	if err != nil {
		return fmt.Errorf("error retrieving subjects: %w", err)
	}

	for sub := range subs {
		err = conn.Send(&pb.LookupSubjectsResponse{
			Subject:    sub.Subject,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(sub.Continuation)},
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

	res, errs, err := s.resourcesUsecase.Get(ctx, req)

	if err != nil {
		return err
	}
	for re := range res {
		err = conn.Send(&pb.LookupResourcesResponse{
			Resource:   re.Resource,
			Pagination: &pb.ResponsePagination{ContinuationToken: string(re.Continuation)},
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
