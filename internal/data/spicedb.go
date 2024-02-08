package data

import (
	apiV1 "ciam-rebac/api/rebac/v1"
	"ciam-rebac/internal/biz"
	"ciam-rebac/internal/conf"
	"context"
	"fmt"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

// SpiceDbRepository .
type SpiceDbRepository struct {
	client *authzed.Client
}

// NewSpiceDbRepository .
func NewSpiceDbRepository(c *conf.Data, logger log.Logger) (*SpiceDbRepository, func(), error) {
	log.NewHelper(logger).Info("creating spicedb connection")

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock()) // TODO: always did it this way with authz. Still the right option?

	token, err := readToken(c.SpiceDb.TokenFile)
	if err != nil {
		err = fmt.Errorf("error extracting token from file: %w", err)
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

func (s *SpiceDbRepository) CreateRelationships(ctx context.Context, rels []*apiV1.Relationship, touch biz.TouchSemantics) error {
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

func (s *SpiceDbRepository) ReadRelationships(ctx context.Context, filter []*apiV1.RelationshipFilter) ([]*apiV1.Relationship, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SpiceDbRepository) DeleteRelationships(ctx context.Context, filter []*apiV1.RelationshipFilter) ([]*apiV1.Relationship, error) {
	//TODO implement me
	panic("implement me")
}

func createSpiceDbRelationshipFilter(filter *apiV1.RelationshipFilter) *v1.RelationshipFilter {
	subject := &v1.SubjectFilter{
		SubjectType:       filter.GetSubjectFilter().GetSubjectType(),
		OptionalSubjectId: filter.GetSubjectFilter().GetSubjectId(),
		OptionalRelation: &v1.SubjectFilter_RelationFilter{
			Relation: filter.GetSubjectFilter().GetRelation(),
		},
	}

	return &v1.RelationshipFilter{
		ResourceType:          filter.GetObjectType(),
		OptionalResourceId:    filter.GetObjectId(),
		OptionalRelation:      filter.GetRelation(),
		OptionalSubjectFilter: subject,
	}
}

func createSpiceDbRelationship(relationship *apiV1.Relationship) *v1.Relationship {
	subject := &v1.SubjectReference{
		Object: &v1.ObjectReference{
			ObjectType: relationship.GetSubject().GetObject().GetType(),
			ObjectId:   relationship.GetSubject().GetObject().GetId(),
		},
		OptionalRelation: relationship.GetSubject().GetRelation(),
	}

	object := &v1.ObjectReference{
		ObjectType: relationship.GetObject().GetType(),
		ObjectId:   relationship.GetObject().GetId(),
	}

	return &v1.Relationship{
		Resource: object,
		Relation: relationship.GetRelation(),
		Subject:  subject,
	}
}

func readToken(file string) (string, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
