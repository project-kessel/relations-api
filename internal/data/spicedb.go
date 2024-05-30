package data

import (
	apiV0 "ciam-rebac/api/relations/v0"
	"ciam-rebac/internal/biz"
	"ciam-rebac/internal/conf"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SpiceDbRepository .
type SpiceDbRepository struct {
	client *authzed.Client
}

// NewSpiceDbRepository .
func NewSpiceDbRepository(c *conf.Data, logger log.Logger) (*SpiceDbRepository, func(), error) {
	log.NewHelper(logger).Info("creating spicedb connection")

	var opts []grpc.DialOption
	opts = append(opts, grpc.EmptyDialOption{})
	//TODO: add a flag to enable/disable grpc.WithBlock

	var token string
	var err error
	if c.SpiceDb.Token != "" {
		token = c.SpiceDb.Token
	} else if c.SpiceDb.TokenFile != "" {
		token, err = readToken(c.SpiceDb.TokenFile)
		if err != nil {
			log.NewHelper(logger).Error(err)
			return nil, nil, err
		}
	}
	if token == "" {
		err := fmt.Errorf("token is empty: %s", token)
		log.NewHelper(logger).Error(err)
		return nil, nil, err
	}

	if !c.SpiceDb.UseTLS {
		opts = append(opts, grpcutil.WithInsecureBearerToken(token))
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		tlsConfig, _ := grpcutil.WithSystemCerts(grpcutil.VerifyCA)
		opts = append(opts, grpcutil.WithBearerToken(token))
		opts = append(opts, tlsConfig)
	}

	client, err := authzed.NewClient(
		c.SpiceDb.Endpoint,
		opts...,
	)

	if err != nil {
		err = fmt.Errorf("error creating spicedb client: %w", err)
		log.NewHelper(logger).Error(err)

		return nil, nil, err
	}

	cleanup := func() {
		log.NewHelper(logger).Info("spicedb connection cleanup requested (nothing to clean up)")
	}

	return &SpiceDbRepository{client}, cleanup, nil
}

func (s *SpiceDbRepository) CreateRelationships(ctx context.Context, rels []*apiV0.Relationship, touch biz.TouchSemantics) error {
	var relationshipUpdates []*v1.RelationshipUpdate

	var operation v1.RelationshipUpdate_Operation
	if touch {
		operation = v1.RelationshipUpdate_OPERATION_TOUCH
	} else {
		operation = v1.RelationshipUpdate_OPERATION_CREATE
	}

	for _, rel := range rels {
		relationshipUpdates = append(relationshipUpdates, &v1.RelationshipUpdate{
			Operation:    operation,
			Relationship: createSpiceDbRelationship(rel),
		})
	}

	_, err := s.client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: relationshipUpdates,
	})

	return err
}

func (s *SpiceDbRepository) ReadRelationships(ctx context.Context, filter *apiV0.RelationTupleFilter) ([]*apiV0.Relationship, error) {
	req := &v1.ReadRelationshipsRequest{RelationshipFilter: createSpiceDbRelationshipFilter(filter)}

	client, err := s.client.ReadRelationships(ctx, req)

	if err != nil {
		return nil, err
	}

	results := make([]*apiV0.Relationship, 0)
	resp, err := client.Recv()
	for err == nil {
		results = append(results, &apiV0.Relationship{
			Resource: &apiV0.ObjectReference{
				Type: spicedbTypeToKesselType(resp.Relationship.Resource.ObjectType),
				Id:   resp.Relationship.Resource.ObjectId,
			},
			Relation: resp.Relationship.Relation,
			Subject: &apiV0.SubjectReference{
				Relation: optionalStringToStringPointer(resp.Relationship.Subject.OptionalRelation),
				Subject: &apiV0.ObjectReference{
					Type: spicedbTypeToKesselType(resp.Relationship.Subject.Object.ObjectType),
					Id:   resp.Relationship.Subject.Object.ObjectId,
				},
			},
		})

		resp, err = client.Recv()
	}

	if !errors.Is(err, io.EOF) {
		return nil, err
	}

	return results, nil
}

func (s *SpiceDbRepository) DeleteRelationships(ctx context.Context, filter *apiV0.RelationTupleFilter) error {
	req := &v1.DeleteRelationshipsRequest{RelationshipFilter: createSpiceDbRelationshipFilter(filter)}

	_, err := s.client.DeleteRelationships(ctx, req)

	// TODO: we have not specified an option in our API to allow partial deletions, so currently it's all or nothing
	if err != nil {
		return err
	}

	return nil
}

func (s *SpiceDbRepository) Check(ctx context.Context, check *apiV0.CheckRequest) (*apiV0.CheckResponse, error) {
	subject := &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: kesselTypeToSpiceDBType(check.GetSubject().GetSubject().Type),
			ObjectId:   check.GetSubject().GetSubject().GetId(),
		},
		OptionalRelation: check.GetSubject().GetRelation(),
	}

	resource := &v1.ObjectReference{
		ObjectType: kesselTypeToSpiceDBType(check.GetResource().GetType()),
		ObjectId:   check.GetResource().GetId(),
	}
	checkResponse, err := s.client.CheckPermission(ctx, &v1.CheckPermissionRequest{
		Resource:   resource,
		Permission: check.GetRelation(),
		Subject:    subject,
	})
	if err != nil {
		log.Errorf("Error check permission %v", err.Error())
		return &apiV0.CheckResponse{Allowed: apiV0.CheckResponse_ALLOWED_UNSPECIFIED}, err
	}

	if checkResponse.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return &apiV0.CheckResponse{Allowed: apiV0.CheckResponse_ALLOWED_TRUE}, nil
	}

	return &apiV0.CheckResponse{Allowed: apiV0.CheckResponse_ALLOWED_FALSE}, nil
}

func createSpiceDbRelationshipFilter(filter *apiV0.RelationTupleFilter) *v1.RelationshipFilter {
	spiceDbRelationshipFilter := &v1.RelationshipFilter{
		ResourceType:       filter.GetResourceType(),
		OptionalResourceId: filter.GetResourceId(),
		OptionalRelation:   filter.GetRelation(),
	}

	if filter.GetSubjectFilter() != nil {
		subjectFilter := &v1.SubjectFilter{
			SubjectType:       filter.GetSubjectFilter().GetSubjectType(),
			OptionalSubjectId: filter.GetSubjectFilter().GetSubjectId(),
		}

		if filter.GetSubjectFilter().GetRelation() != "" {
			subjectFilter.OptionalRelation = &v1.SubjectFilter_RelationFilter{
				Relation: filter.GetSubjectFilter().GetRelation(),
			}
		}

		spiceDbRelationshipFilter.OptionalSubjectFilter = subjectFilter
	}

	return spiceDbRelationshipFilter
}

func spicedbTypeToKesselType(spicedbType string) *apiV0.ObjectType {
	kesselType := &apiV0.ObjectType{}

	parts := strings.Split(spicedbType, "/")
	switch len(parts) {
	case 1:
		kesselType.Type = parts[0]
	case 2:
		kesselType.Namespace = parts[0]
		kesselType.Type = parts[1]
	default:
		return nil //?? Error?
	}

	return kesselType
}

func kesselTypeToSpiceDBType(kesselType *apiV0.ObjectType) string {
	if kesselType.Namespace != "" {
		return fmt.Sprintf("%s/%s", kesselType.Namespace, kesselType.Type)
	}

	return kesselType.Type
}

func optionalStringToStringPointer(optional string) *string {
	if optional == "" {
		return nil
	}

	return &optional
}

func createSpiceDbRelationship(relationship *apiV0.Relationship) *v1.Relationship {
	subject := &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: kesselTypeToSpiceDBType(relationship.GetSubject().GetSubject().GetType()),
			ObjectId:   relationship.GetSubject().GetSubject().GetId(),
		},
		OptionalRelation: relationship.GetSubject().GetRelation(),
	}

	object := &v1.ObjectReference{
		ObjectType: kesselTypeToSpiceDBType(relationship.GetResource().GetType()),
		ObjectId:   relationship.GetResource().GetId(),
	}

	return &v1.Relationship{
		Resource: object,
		Relation: relationship.GetRelation(),
		Subject:  subject,
	}
}

func readToken(file string) (string, error) {
	isFileExist := checkFileExists(file)
	if !isFileExist {
		return file, errors.New("file doesn't exist")
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}
