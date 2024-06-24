package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	apiV0 "github.com/project-kessel/relations-api/api/relations/v0"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"

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

	// wait upto 30 seconds to confirm spicedb is connected
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err = client.ReadSchema(ctx, &v1.ReadSchemaRequest{}, grpc.WaitForReady(true))
	if err != nil {
		return nil, nil, fmt.Errorf("error trying to reach SpiceDB: %w", err)
	}

	select {
	case <-ctx.Done():
		log.NewHelper(logger).Errorf("timeout exceeded waiting for spicedb %w", ctx.Err())
	default:
		log.NewHelper(logger).Infof("Successfully connected to SpiceDB")
	}

	cleanup := func() {
		log.NewHelper(logger).Info("spicedb connection cleanup requested (nothing to clean up)")
	}

	return &SpiceDbRepository{client}, cleanup, nil
}

func (s *SpiceDbRepository) LookupSubjects(ctx context.Context, subject_type *apiV0.ObjectType, subject_relation, relation string, object *apiV0.ObjectReference, limit uint32, continuation biz.ContinuationToken) (chan *biz.SubjectResult, chan error, error) {
	var cursor *v1.Cursor = nil
	if continuation != "" {
		cursor = &v1.Cursor{
			Token: string(continuation),
		}
	}

	client, err := s.client.LookupSubjects(ctx, &v1.LookupSubjectsRequest{
		Resource: &v1.ObjectReference{
			ObjectType: kesselTypeToSpiceDBType(object.Type),
			ObjectId:   object.Id,
		},
		Permission:              relation,
		SubjectObjectType:       kesselTypeToSpiceDBType(subject_type),
		WildcardOption:          v1.LookupSubjectsRequest_WILDCARD_OPTION_EXCLUDE_WILDCARDS,
		OptionalSubjectRelation: subject_relation,
		OptionalConcreteLimit:   limit,
		OptionalCursor:          cursor,
	})

	if err != nil {
		return nil, nil, err
	}

	subjects := make(chan *biz.SubjectResult)
	errs := make(chan error, 1)

	go func() {
		for {
			msg, err := client.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errs <- err
				}
				close(errs)
				close(subjects)
				return
			}

			continuation := biz.ContinuationToken("")
			if msg.AfterResultCursor != nil {
				continuation = biz.ContinuationToken(msg.AfterResultCursor.Token)
			}

			subj := msg.GetSubject()
			subjects <- &biz.SubjectResult{
				Subject: &apiV0.SubjectReference{
					Subject: &apiV0.ObjectReference{
						Type: subject_type,
						Id:   subj.SubjectObjectId,
					},
				},
				Continuation: continuation,
			}
		}
	}()

	return subjects, errs, nil
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

func (s *SpiceDbRepository) ReadRelationships(ctx context.Context, filter *apiV0.RelationTupleFilter, limit uint32, continuation biz.ContinuationToken) (chan *biz.RelationshipResult, chan error, error) {
	var cursor *v1.Cursor = nil
	if continuation != "" {
		cursor = &v1.Cursor{
			Token: string(continuation),
		}
	}
	client, err := s.client.ReadRelationships(ctx, &v1.ReadRelationshipsRequest{
		RelationshipFilter: createSpiceDbRelationshipFilter(filter),
		OptionalLimit:      limit,
		OptionalCursor:     cursor,
	})

	if err != nil {
		return nil, nil, err
	}

	relationshipTuples := make(chan *biz.RelationshipResult)
	errs := make(chan error, 1)

	go func() {
		for {
			msg, err := client.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errs <- err
				}
				close(errs)
				close(relationshipTuples)
				return
			}

			continuation := biz.ContinuationToken("")
			if msg.AfterResultCursor != nil {
				continuation = biz.ContinuationToken(msg.AfterResultCursor.Token)
			}

			spiceDbRel := msg.GetRelationship()
			relationshipTuples <- &biz.RelationshipResult{
				Relationship: &apiV0.Relationship{
					Resource: &apiV0.ObjectReference{
						Type: spicedbTypeToKesselType(spiceDbRel.Resource.ObjectType),
						Id:   spiceDbRel.Resource.ObjectId,
					},
					Relation: msg.Relationship.Relation,
					Subject: &apiV0.SubjectReference{
						Relation: optionalStringToStringPointer(spiceDbRel.Subject.OptionalRelation),
						Subject: &apiV0.ObjectReference{
							Type: spicedbTypeToKesselType(spiceDbRel.Subject.Object.ObjectType),
							Id:   spiceDbRel.Subject.Object.ObjectId,
						},
					},
				},
				Continuation: continuation,
			}
		}
	}()

	return relationshipTuples, errs, nil
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
		kesselType.Name = parts[0]
	case 2:
		kesselType.Namespace = parts[0]
		kesselType.Name = parts[1]
	default:
		return nil //?? Error?
	}

	return kesselType
}

func kesselTypeToSpiceDBType(kesselType *apiV0.ObjectType) string {
	if kesselType.Namespace != "" {
		return fmt.Sprintf("%s/%s", kesselType.Namespace, kesselType.Name)
	}

	return kesselType.Name
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
