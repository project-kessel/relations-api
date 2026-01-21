package biz

import (
	"context"
	"testing"

	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// DummyZanzibar is a fake implementation of ZanzibarRepository for testing
type DummyZanzibar struct {
	subjects       []*SubjectResult
	resources      []*ResourceResult
	subjectsError  error
	resourcesError error
	capturedLimit  uint32
}

func (dz *DummyZanzibar) Check(ctx context.Context, request *v1beta1.CheckRequest) (*v1beta1.CheckResponse, error) {
	return nil, nil
}

func (dz *DummyZanzibar) CheckForUpdate(ctx context.Context, request *v1beta1.CheckForUpdateRequest) (*v1beta1.CheckForUpdateResponse, error) {
	return nil, nil
}

func (dz *DummyZanzibar) CheckBulk(ctx context.Context, request *v1beta1.CheckBulkRequest) (*v1beta1.CheckBulkResponse, error) {
	return nil, nil
}

func (dz *DummyZanzibar) CreateRelationships(ctx context.Context, rels []*v1beta1.Relationship, touch TouchSemantics, fencing *v1beta1.FencingCheck) (*v1beta1.CreateTuplesResponse, error) {
	return nil, nil
}

func (dz *DummyZanzibar) ReadRelationships(ctx context.Context, filter *v1beta1.RelationTupleFilter, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *RelationshipResult, chan error, error) {
	return nil, nil, nil
}

func (dz *DummyZanzibar) DeleteRelationships(ctx context.Context, filter *v1beta1.RelationTupleFilter, fencing *v1beta1.FencingCheck) (*v1beta1.DeleteTuplesResponse, error) {
	return nil, nil
}

func (dz *DummyZanzibar) LookupSubjects(ctx context.Context, subjectType *v1beta1.ObjectType, subject_relation, relation string, resource *v1beta1.ObjectReference, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *SubjectResult, chan error, error) {
	// Capture the limit for assertions
	dz.capturedLimit = limit

	subjectsChan := make(chan *SubjectResult)
	errsChan := make(chan error, 1)

	go func() {
		defer close(subjectsChan)
		defer close(errsChan)

		if dz.subjectsError != nil {
			errsChan <- dz.subjectsError
			return
		}

		// Respect the limit when sending results
		count := uint32(0)
		for _, subject := range dz.subjects {
			// If limit is 0, no limit (send all). Otherwise, respect the limit.
			if limit > 0 && count >= limit {
				break
			}
			subjectsChan <- subject
			count++
		}
	}()

	return subjectsChan, errsChan, nil
}

func (dz *DummyZanzibar) LookupResources(ctx context.Context, resource_type *v1beta1.ObjectType, relation string, subject *v1beta1.SubjectReference, limit uint32, continuation ContinuationToken, consistency *v1beta1.Consistency) (chan *ResourceResult, chan error, error) {
	// Capture the limit for assertions
	dz.capturedLimit = limit

	resourcesChan := make(chan *ResourceResult)
	errsChan := make(chan error, 1)

	go func() {
		defer close(resourcesChan)
		defer close(errsChan)

		if dz.resourcesError != nil {
			errsChan <- dz.resourcesError
			return
		}

		// Respect the limit when sending results
		count := uint32(0)
		for _, resource := range dz.resources {
			// If limit is 0, no limit (send all). Otherwise, respect the limit.
			if limit > 0 && count >= limit {
				break
			}
			resourcesChan <- resource
			count++
		}
	}()

	return resourcesChan, errsChan, nil
}

func (dz *DummyZanzibar) IsBackendAvailable() error {
	return nil
}

func (dz *DummyZanzibar) ImportBulkTuples(stream grpc.ClientStreamingServer[v1beta1.ImportBulkTuplesRequest, v1beta1.ImportBulkTuplesResponse]) error {
	return nil
}

func (dz *DummyZanzibar) AcquireLock(ctx context.Context, lockId string) (*v1beta1.AcquireLockResponse, error) {
	return nil, nil
}

// Helper to create test subjects
func createTestSubject(id string) *SubjectResult {
	return &SubjectResult{
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: id,
			},
		},
		Continuation:     "",
		ConsistencyToken: &v1beta1.ConsistencyToken{Token: "token"},
	}
}

// Helper to create test resources
func createTestResource(id string) *ResourceResult {
	return &ResourceResult{
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "widget",
			},
			Id: id,
		},
		Continuation:     "",
		ConsistencyToken: &v1beta1.ConsistencyToken{Token: "token"},
	}
}

// Helper to collect subjects from channels
func collectSubjects(t *testing.T, subjects chan *SubjectResult, errs chan error) []*SubjectResult {
	var results []*SubjectResult
	for {
		select {
		case subj, ok := <-subjects:
			if !ok {
				return results
			}
			results = append(results, subj)
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatalf("Error receiving subjects: %v", err)
			}
		}
	}
}

// Helper to collect resources from channels
func collectResources(t *testing.T, resources chan *ResourceResult, errs chan error) []*ResourceResult {
	var results []*ResourceResult
	for {
		select {
		case res, ok := <-resources:
			if !ok {
				return results
			}
			results = append(results, res)
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatalf("Error receiving resources: %v", err)
			}
		}
	}
}

// Tests for GetSubjectsUsecase

func TestGetSubjectsUsecase_Get_WithNoLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		subjects: []*SubjectResult{
			createTestSubject("alice"),
			createTestSubject("bob"),
		},
	}

	usecase := NewGetSubjectsUseCase(dummy)

	req := &v1beta1.LookupSubjectsRequest{
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "workspace",
			},
			Id: "test",
		},
		Relation: "view_widget",
		SubjectType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "principal",
		},
		// No Pagination set - should use default limit of 0
	}

	subjects, errs, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	results := collectSubjects(t, subjects, errs)

	assert.Equal(t, uint32(0), dummy.capturedLimit, "should pass limit of 0 when no pagination specified")
	assert.Len(t, results, 2)
}

func TestGetSubjectsUsecase_Get_WithPaginationLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		subjects: []*SubjectResult{
			createTestSubject("alice"),
			createTestSubject("bob"),
			createTestSubject("charlie"),
		},
	}

	usecase := NewGetSubjectsUseCase(dummy)

	req := &v1beta1.LookupSubjectsRequest{
		Resource: &v1beta1.ObjectReference{
			Type: &v1beta1.ObjectType{
				Namespace: "rbac",
				Name:      "workspace",
			},
			Id: "test",
		},
		Relation: "view_widget",
		SubjectType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "principal",
		},
		Pagination: &v1beta1.RequestPagination{
			Limit: 2,
		},
	}

	subjects, errs, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	results := collectSubjects(t, subjects, errs)

	assert.Equal(t, uint32(2), dummy.capturedLimit, "should pass the requested limit")
	assert.Len(t, results, 2, "should only return 2 subjects even though 3 are available")
}

// Tests for GetResourcesUsecase

func TestGetResourcesUsecase_Get_WithNoLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		resources: []*ResourceResult{
			createTestResource("widget1"),
			createTestResource("widget2"),
		},
	}

	usecase := NewGetResourcesUseCase(dummy)

	req := &v1beta1.LookupResourcesRequest{
		ResourceType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "widget",
		},
		Relation: "view",
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "alice",
			},
		},
		// No Pagination set - should use MaxStreamingCount
	}

	resources, errs, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	results := collectResources(t, resources, errs)

	assert.Equal(t, uint32(MaxStreamingCount), dummy.capturedLimit, "should use MaxStreamingCount when no pagination specified")
	assert.Len(t, results, 2)
}

func TestGetResourcesUsecase_Get_WithPaginationLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		resources: []*ResourceResult{
			createTestResource("widget1"),
			createTestResource("widget2"),
			createTestResource("widget3"),
		},
	}

	usecase := NewGetResourcesUseCase(dummy)

	req := &v1beta1.LookupResourcesRequest{
		ResourceType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "widget",
		},
		Relation: "view",
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "alice",
			},
		},
		Pagination: &v1beta1.RequestPagination{
			Limit: 2,
		},
	}

	resources, errs, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	results := collectResources(t, resources, errs)

	assert.Equal(t, uint32(2), dummy.capturedLimit, "should pass the requested limit when less than MaxStreamingCount")
	assert.Len(t, results, 2, "should only return 2 resources even though 3 are available")
}

func TestGetResourcesUsecase_Get_WithLimitGreaterThanMax(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		resources: []*ResourceResult{
			createTestResource("widget1"),
		},
	}

	usecase := NewGetResourcesUseCase(dummy)

	req := &v1beta1.LookupResourcesRequest{
		ResourceType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "widget",
		},
		Relation: "view",
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "alice",
			},
		},
		Pagination: &v1beta1.RequestPagination{
			Limit: MaxStreamingCount + 100,
		},
	}

	_, _, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	assert.Equal(t, uint32(MaxStreamingCount), dummy.capturedLimit, "should cap limit at MaxStreamingCount")
}

func TestGetResourcesUsecase_Get_WithZeroLimit(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dummy := &DummyZanzibar{
		resources: []*ResourceResult{
			createTestResource("widget1"),
			createTestResource("widget2"),
		},
	}

	usecase := NewGetResourcesUseCase(dummy)

	req := &v1beta1.LookupResourcesRequest{
		ResourceType: &v1beta1.ObjectType{
			Namespace: "rbac",
			Name:      "widget",
		},
		Relation: "view",
		Subject: &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Namespace: "rbac",
					Name:      "principal",
				},
				Id: "alice",
			},
		},
		Pagination: &v1beta1.RequestPagination{
			Limit: 0,
		},
	}

	resources, errs, err := usecase.Get(ctx, req)
	assert.NoError(t, err)

	results := collectResources(t, resources, errs)

	assert.Equal(t, uint32(0), dummy.capturedLimit, "should pass limit of 0 when explicitly requested")
	assert.Len(t, results, 2, "should return all resources when limit is 0")
}
