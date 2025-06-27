package biz

import (
	"context"
	"fmt"

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
	newToken, err := rc.repo.AcquireLock(ctx, req.Identifier, req.GetExistingToken())
	if err != nil {
		return nil, fmt.Errorf("could not acquire lock: %w", err)
	}

	return &v1beta1.AcquireLockResponse{
		NewToken: newToken,
	}, nil
}
