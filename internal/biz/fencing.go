package biz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	v1beta1 "github.com/project-kessel/relations-api/api/kessel/relations/v1beta1"
)

type AcquireLockUsecase struct {
	repo ZanzibarRepository
}

func NewAcquireLockUsecase(repo ZanzibarRepository) *AcquireLockUsecase {
	return &AcquireLockUsecase{
		repo: repo,
	}
}

func (rc *AcquireLockUsecase) Get(ctx context.Context, req *v1beta1.AcquireLockRequest) (*v1beta1.AcquireLockResponse, error) {
	var oldLock *string

	lock := "lock"
	version := "version"

	filter := &v1beta1.RelationTupleFilter{
		ResourceType: &lock,
		ResourceId:   &req.Identifier,
		Relation:     &version,
	}

	resultsChan, errChan, err := rc.repo.ReadRelationships(ctx, filter, 1, "", nil)
	if err != nil {
		return nil, fmt.Errorf("could not read existing token: %w", err)
	}

	select {
	case result, ok := <-resultsChan:
		if ok && result.Relationship.Subject.GetSubject() != nil {
			oldLock = &result.Relationship.Subject.Subject.Id
		}
	case err, ok := <-errChan:
		if ok {
			return nil, fmt.Errorf("error reading existing token: %w", err)
		}
	}

	newLock := uuid.New().String()
	updates := []*ExperimentalWrite{}

	if oldLock != nil {
		updates = append(updates, &ExperimentalWrite{
			Operation: OperationDelete,
			Relationship: &v1beta1.Relationship{
				Resource: &v1beta1.ObjectReference{
					Type: &v1beta1.ObjectType{
						Name: "lock",
					},
					Id: req.Identifier,
				},
				Relation: "version",
				Subject: &v1beta1.SubjectReference{
					Subject: &v1beta1.ObjectReference{
						Type: &v1beta1.ObjectType{
							Name: "lockversion",
						},
						Id: *oldLock,
					},
				},
			},
		})
	}

	updates = append(updates, &ExperimentalWrite{
		Operation: OperationCreate,
		Relationship: &v1beta1.Relationship{
			Resource: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Name: "lock",
				},
				Id: req.Identifier,
			},
			Relation: "version",
			Subject: &v1beta1.SubjectReference{
				Subject: &v1beta1.ObjectReference{
					Type: &v1beta1.ObjectType{
						Name: "lockversion",
					},
					Id: newLock,
				},
			},
		},
	})

	_, err = rc.repo.ExperimentalWrite(ctx, updates)
	if err != nil {
		return nil, fmt.Errorf("could not write lock: %w", err)
	}

	return &v1beta1.AcquireLockResponse{
		NewLock: newLock,
	}, nil
}
