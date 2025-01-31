package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	apiV1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/conf"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// SpiceDbRepository .
type SpiceDbRepository struct {
	client          *authzed.Client
	healthClient    grpc_health_v1.HealthClient
	schemaFilePath  string
	isInitialized   bool
	fullyConsistent bool
}

const (
	relationPrefix = "t_"
)

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
		token, err = readFile(c.SpiceDb.TokenFile)
		if err != nil {
			log.NewHelper(logger).Error(err)
			return nil, nil, fmt.Errorf("error creating spicedb client: error loading token file: %w", err)
		}
	}
	if token == "" {
		return nil, nil, fmt.Errorf("error creating spicedb client: token is empty")
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
		return nil, nil, fmt.Errorf("error creating spicedb client: %w", err)
	}

	// Create health client for readyz
	conn, err := grpc.NewClient(
		c.SpiceDb.Endpoint,
		opts...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating grpc health client: %w", err)
	}
	healthClient := grpc_health_v1.NewHealthClient(conn)

	cleanup := func() {
		log.NewHelper(logger).Info("spicedb connection cleanup requested (nothing to clean up)")
	}

	return &SpiceDbRepository{client, healthClient, c.SpiceDb.SchemaFile, false, c.SpiceDb.FullyConsistent}, cleanup, nil
}

func (s *SpiceDbRepository) initialize() error {
	if s.isInitialized {
		return nil
	}

	schema, err := readFile(s.schemaFilePath)
	if err != nil {
		return fmt.Errorf("failed to load schema file: %w", err)
	}

	_, err = s.client.WriteSchema(context.TODO(), &v1.WriteSchemaRequest{
		Schema: schema,
	})

	if err != nil {
		return err
	}

	s.isInitialized = true
	return nil
}

func (s *SpiceDbRepository) LookupSubjects(ctx context.Context, subject_type *apiV1beta1.ObjectType, subject_relation, relation string, object *apiV1beta1.ObjectReference, limit uint32, continuation biz.ContinuationToken, zookie *apiV1beta1.Zookie) (chan *biz.SubjectResult, chan error, error) {
	if err := s.initialize(); err != nil {
		return nil, nil, err
	}

	var cursor *v1.Cursor = nil
	if continuation != "" {
		cursor = &v1.Cursor{
			Token: string(continuation),
		}
	}

	req := &v1.LookupSubjectsRequest{
		Consistency: s.determineConsistency(zookie, false),
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
	}

	client, err := s.client.LookupSubjects(ctx, req)

	if err != nil {
		return nil, nil, fmt.Errorf("error invoking LookupSubjects in SpiceDB: %w", err)
	}

	subjects := make(chan *biz.SubjectResult)
	errs := make(chan error, 1)

	go func() {
		for {
			msg, err := client.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errs <- fmt.Errorf("error receiving subject from SpiceDB: %w", err)
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
				Subject: &apiV1beta1.SubjectReference{
					Subject: &apiV1beta1.ObjectReference{
						Type: subject_type,
						Id:   subj.SubjectObjectId,
					},
				},
				Continuation: continuation,
				Zookie:       &apiV1beta1.Zookie{Token: msg.GetLookedUpAt().GetToken()},
			}
		}
	}()

	return subjects, errs, nil
}

func (s *SpiceDbRepository) LookupResources(ctx context.Context, resouce_type *apiV1beta1.ObjectType, relation string, subject *apiV1beta1.SubjectReference, limit uint32, continuation biz.ContinuationToken, zookie *apiV1beta1.Zookie) (chan *biz.ResourceResult, chan error, error) {
	if err := s.initialize(); err != nil {
		return nil, nil, err
	}

	var cursor *v1.Cursor = nil
	if continuation != "" {
		cursor = &v1.Cursor{
			Token: string(continuation),
		}
	}
	client, err := s.client.LookupResources(ctx, &v1.LookupResourcesRequest{
		Consistency:        s.determineConsistency(zookie, false),
		ResourceObjectType: kesselTypeToSpiceDBType(resouce_type),
		Permission:         relation,
		Subject: &v1.SubjectReference{
			OptionalRelation: optionalStringPointerToString(subject.Relation),
			Object: &v1.ObjectReference{
				ObjectType: kesselTypeToSpiceDBType(subject.Subject.Type),
				ObjectId:   subject.Subject.Id,
			},
		},
		OptionalLimit:  limit,
		OptionalCursor: cursor,
	})
	if err != nil {
		return nil, nil, err
	}

	resources := make(chan *biz.ResourceResult)
	errs := make(chan error, 1)

	go func() {
		for {
			msg, err := client.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					errs <- err
				}
				close(errs)
				close(resources)
				return
			}

			continuation := biz.ContinuationToken("")
			if msg.AfterResultCursor != nil {
				continuation = biz.ContinuationToken(msg.AfterResultCursor.Token)
			}

			resId := msg.GetResourceObjectId()
			resources <- &biz.ResourceResult{
				Resource: &apiV1beta1.ObjectReference{
					Type: resouce_type,
					Id:   resId,
				},
				Continuation: continuation,
				Zookie:       &apiV1beta1.Zookie{Token: msg.GetLookedUpAt().GetToken()},
			}
		}
	}()
	return resources, errs, nil
}

func (s *SpiceDbRepository) ImportBulkTuples(stream grpc.ClientStreamingServer[apiV1beta1.ImportBulkTuplesRequest, apiV1beta1.ImportBulkTuplesResponse]) error {
	if err := s.initialize(); err != nil {
		return err
	}

	var totalImported uint64
	client, err := s.client.ImportBulkRelationships(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create SpiceDB client: %w", err)
	}

	for {
		req, streamErr := stream.Recv()
		if streamErr != nil {
			if req == nil && errors.Is(streamErr, io.EOF) {
				if res, closeErr := client.CloseAndRecv(); closeErr != nil {
					return fmt.Errorf("error receiving response from Spicedb for bulkimport request: %w", closeErr)
				} else {
					log.Infof("total number of relationships loaded: %d", res.NumLoaded)
					totalImported = res.NumLoaded
					return stream.SendAndClose(&apiV1beta1.ImportBulkTuplesResponse{NumImported: totalImported})
				}
			}
			return streamErr
		}
		inputRelationships := (*req).Tuples
		batch := []*v1.Relationship{}
		for _, tuple := range inputRelationships {
			tuple.Relation = addRelationPrefix(tuple.Relation, relationPrefix)
			batch = append(batch, createSpiceDbRelationship(tuple))
		}
		if err = client.Send((*v1.ImportBulkRelationshipsRequest)(&v1.BulkImportRelationshipsRequest{
			Relationships: batch,
		})); err != nil {
			if !errors.Is(err, io.EOF) {
				return fmt.Errorf("failed to send bulkimport request: %w", err)
			}
			return err
		}
	}

}

func (s *SpiceDbRepository) CreateRelationships(ctx context.Context, rels []*apiV1beta1.Relationship, touch biz.TouchSemantics) (*apiV1beta1.CreateTuplesResponse, error) {
	if err := s.initialize(); err != nil {
		return nil, err
	}

	var relationshipUpdates []*v1.RelationshipUpdate

	var operation v1.RelationshipUpdate_Operation
	if touch {
		operation = v1.RelationshipUpdate_OPERATION_TOUCH
	} else {
		operation = v1.RelationshipUpdate_OPERATION_CREATE
	}

	for _, rel := range rels {
		rel.Relation = addRelationPrefix(rel.Relation, relationPrefix)
		// subject relations are intentionally not prefixed here
		// bc we want to reference the corresponding permission

		relationshipUpdates = append(relationshipUpdates, &v1.RelationshipUpdate{
			Operation:    operation,
			Relationship: createSpiceDbRelationship(rel),
		})
	}

	resp, err := s.client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
		Updates: relationshipUpdates,
	})

	if err != nil {
		return nil, fmt.Errorf("error writing relationships to SpiceDB: %w", err)
	}

	return &apiV1beta1.CreateTuplesResponse{CreatedAt: &apiV1beta1.Zookie{Token: resp.GetWrittenAt().GetToken()}}, nil
}

func (s *SpiceDbRepository) ReadRelationships(ctx context.Context, filter *apiV1beta1.RelationTupleFilter, limit uint32, continuation biz.ContinuationToken, zookie *apiV1beta1.Zookie) (chan *biz.RelationshipResult, chan error, error) {
	if err := s.initialize(); err != nil {
		return nil, nil, err
	}

	var cursor *v1.Cursor = nil
	if continuation != "" {
		cursor = &v1.Cursor{
			Token: string(continuation),
		}
	}

	if filter.GetRelation() != "" {
		// subject relations are intentionally not prefixed here
		// bc we want to reference the corresponding permission
		tempRelation := addRelationPrefix(filter.GetRelation(), relationPrefix)
		filter.Relation = &tempRelation
	}

	relationshipFilter, err := createSpiceDbRelationshipFilter(filter)

	if err != nil {
		return nil, nil, kerrors.BadRequest("SpiceDb request validation", err.Error()).WithCause(err)
	}

	req := &v1.ReadRelationshipsRequest{
		Consistency:        s.determineConsistency(zookie, false),
		RelationshipFilter: relationshipFilter,
		OptionalLimit:      limit,
		OptionalCursor:     cursor,
	}

	client, err := s.client.ReadRelationships(ctx, req)

	if err != nil {
		return nil, nil, fmt.Errorf("error invoking WriteRelationships in SpiceDB: %w", err)
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
				Relationship: &apiV1beta1.Relationship{
					Resource: &apiV1beta1.ObjectReference{
						Type: spicedbTypeToKesselType(spiceDbRel.Resource.ObjectType),
						Id:   spiceDbRel.Resource.ObjectId,
					},
					Relation: strings.TrimPrefix(msg.Relationship.Relation, relationPrefix),
					Subject: &apiV1beta1.SubjectReference{
						Relation: optionalStringToStringPointer(spiceDbRel.Subject.OptionalRelation),
						Subject: &apiV1beta1.ObjectReference{
							Type: spicedbTypeToKesselType(spiceDbRel.Subject.Object.ObjectType),
							Id:   spiceDbRel.Subject.Object.ObjectId,
						},
					},
				},
				Continuation: continuation,
				Zookie:       &apiV1beta1.Zookie{Token: msg.ReadAt.GetToken()},
			}
		}
	}()

	return relationshipTuples, errs, nil
}

func (s *SpiceDbRepository) DeleteRelationships(ctx context.Context, filter *apiV1beta1.RelationTupleFilter) (*apiV1beta1.DeleteTuplesResponse, error) {
	if err := s.initialize(); err != nil {
		return nil, err
	}

	if filter.GetRelation() != "" {
		tempRelation := addRelationPrefix(filter.GetRelation(), relationPrefix)
		filter.Relation = &tempRelation
	}

	relationshipFilter, err := createSpiceDbRelationshipFilter(filter)

	if err != nil {
		return nil, kerrors.BadRequest("SpiceDb request validation", err.Error()).WithCause(err)
	}

	req := &v1.DeleteRelationshipsRequest{RelationshipFilter: relationshipFilter}

	resp, err := s.client.DeleteRelationships(ctx, req)

	// TODO: we have not specified an option in our API to allow partial deletions, so currently it's all or nothing
	if err != nil {
		return nil, fmt.Errorf("error invoking DeleteRelationships in SpiceDB %w", err)
	}

	return &apiV1beta1.DeleteTuplesResponse{DeletedAt: &apiV1beta1.Zookie{Token: resp.GetDeletedAt().GetToken()}}, nil
}

func (s *SpiceDbRepository) Check(ctx context.Context, check *apiV1beta1.CheckRequest) (*apiV1beta1.CheckResponse, error) {
	if err := s.initialize(); err != nil {
		return nil, err
	}

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
	req := &v1.CheckPermissionRequest{
		Consistency: s.determineConsistency(check.Zookie, check.GetFullyConsistent()),
		Resource:    resource,
		Permission:  check.GetRelation(),
		Subject:     subject,
	}
	checkResponse, err := s.client.CheckPermission(ctx, req)
	if err != nil {
		return &apiV1beta1.CheckResponse{Allowed: apiV1beta1.CheckResponse_ALLOWED_UNSPECIFIED}, fmt.Errorf("error invoking CheckPermission in SpiceDB: %w", err)
	}

	if checkResponse.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return &apiV1beta1.CheckResponse{
			Allowed:   apiV1beta1.CheckResponse_ALLOWED_TRUE,
			CheckedAt: &apiV1beta1.Zookie{Token: checkResponse.GetCheckedAt().GetToken()},
		}, nil
	}

	return &apiV1beta1.CheckResponse{
		Allowed:   apiV1beta1.CheckResponse_ALLOWED_FALSE,
		CheckedAt: &apiV1beta1.Zookie{Token: checkResponse.GetCheckedAt().GetToken()},
	}, nil
}

func (s *SpiceDbRepository) IsBackendAvailable() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("timeout connecting to backend")
	default:
		switch resp.Status {
		case grpc_health_v1.HealthCheckResponse_NOT_SERVING, grpc_health_v1.HealthCheckResponse_SERVICE_UNKNOWN:
			return fmt.Errorf("error connecting to backend: %v", resp.Status.String())
		case grpc_health_v1.HealthCheckResponse_SERVING:
			return nil
		}
	}
	return fmt.Errorf("error connecting to backend")
}

func createSpiceDbRelationshipFilter(filter *apiV1beta1.RelationTupleFilter) (*v1.RelationshipFilter, error) {
	// spicedb specific internal validation to reflect spicedb limitations whereby namespace and objectType must be both
	// be set if either of them is set in a filter
	if filter.GetResourceNamespace() != "" && filter.GetResourceType() == "" {
		return nil, fmt.Errorf("due to a spicedb limitation, if resource namespace is specified then resource type must also be specified")
	}
	if filter.GetResourceNamespace() == "" && filter.GetResourceType() != "" {
		return nil, fmt.Errorf("due to a spicedb limitation, if resource type is specified then resource namespace must also be specified")
	}

	resourceType := &apiV1beta1.ObjectType{Namespace: filter.GetResourceNamespace(), Name: filter.GetResourceType()}
	spiceDbRelationshipFilter := &v1.RelationshipFilter{
		ResourceType:       kesselTypeToSpiceDBType(resourceType),
		OptionalResourceId: filter.GetResourceId(),
		OptionalRelation:   filter.GetRelation(),
	}

	if filter.GetSubjectFilter() != nil {
		subjectFilter := filter.GetSubjectFilter()

		if subjectFilter.GetSubjectNamespace() != "" && subjectFilter.GetSubjectType() == "" {
			return nil, fmt.Errorf("due to a spicedb limitation, if subject namespace is specified in subjectFilter then subject type must also be specified")
		}
		if subjectFilter.GetSubjectNamespace() == "" && subjectFilter.GetSubjectType() != "" {
			return nil, fmt.Errorf("due to a spicedb limitation, if subject type is specified in subjectFilter then subject namespace must also be specified")
		}

		subjectType := &apiV1beta1.ObjectType{Namespace: subjectFilter.GetSubjectNamespace(), Name: subjectFilter.GetSubjectType()}
		spiceDbSubjectFilter := &v1.SubjectFilter{
			SubjectType:       kesselTypeToSpiceDBType(subjectType),
			OptionalSubjectId: subjectFilter.GetSubjectId(),
		}

		if subjectFilter.GetRelation() != "" {
			spiceDbSubjectFilter.OptionalRelation = &v1.SubjectFilter_RelationFilter{
				Relation: subjectFilter.GetRelation(),
			}
		}

		spiceDbRelationshipFilter.OptionalSubjectFilter = spiceDbSubjectFilter
	}

	return spiceDbRelationshipFilter, nil
}

func spicedbTypeToKesselType(spicedbType string) *apiV1beta1.ObjectType {
	kesselType := &apiV1beta1.ObjectType{}

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

func kesselTypeToSpiceDBType(kesselType *apiV1beta1.ObjectType) string {
	if kesselType.Namespace != "" {
		return fmt.Sprintf("%s/%s", kesselType.Namespace, kesselType.Name)
	}

	return kesselType.Name
}

func optionalStringPointerToString(optional *string) string {
	if optional == nil {
		return ""
	}
	return *optional
}

func optionalStringToStringPointer(optional string) *string {
	if optional == "" {
		return nil
	}

	return &optional
}

func addRelationPrefix(relation, prefix string) string {
	if !strings.HasPrefix(relation, prefix) {
		return prefix + relation
	}
	return relation
}

func createSpiceDbRelationship(relationship *apiV1beta1.Relationship) *v1.Relationship {
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

func readFile(file string) (string, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (s *SpiceDbRepository) determineConsistency(zookie *apiV1beta1.Zookie, fullyConsistent bool) *v1.Consistency {
	if s.fullyConsistent || fullyConsistent {
		// will ensure that all data used is fully consistent with the latest data available within the SpiceDB datastore.
		return &v1.Consistency{Requirement: &v1.Consistency_FullyConsistent{FullyConsistent: true}}
	}

	if zookie != nil {
		return &v1.Consistency{
			Requirement: &v1.Consistency_AtLeastAsFresh{
				AtLeastAsFresh: &v1.ZedToken{Token: zookie.GetToken()},
			},
		}
	}
	// Default consistency for read APIs is minimize_latency
	return &v1.Consistency{
		Requirement: &v1.Consistency_MinimizeLatency{
			MinimizeLatency: true,
		},
	}
}
