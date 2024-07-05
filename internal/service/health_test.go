package service

import (
	"context"
	"fmt"
	"testing"

	pb "github.com/project-kessel/relations-api/api/kessel/relations/v1"
	"github.com/project-kessel/relations-api/internal/biz"
	"github.com/project-kessel/relations-api/internal/data"

	"github.com/stretchr/testify/assert"
)

func TestHealthService_GetLivez(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	service := createHealthService(spicedb)
	resp, err := service.GetLivez(ctx, &pb.GetLivezRequest{})

	assert.NoError(t, err)
	assert.Equal(t, resp, &pb.GetLivezResponse{Status: "OK", Code: 200})
}

func TestHealthService_GetReadyz_SpiceDBAvailable(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	service := createHealthService(spicedb)
	resp, err := service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, resp, &pb.GetReadyzResponse{Status: "OK", Code: 200})
}

func TestHealthService_GetReadyz_SpiceDBUnavailable(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	d := &DummyZanzibar{}
	service := createDummyHealthService(d)
	resp, err := service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, resp, &pb.GetReadyzResponse{Status: "Unavailable", Code: 503})
}

type DummyZanzibar struct {
	biz.ZanzibarRepository
}

func (dz *DummyZanzibar) IsBackendAvailable() error {
	return fmt.Errorf("Unavailable")
}

func createDummyHealthService(d *DummyZanzibar) *HealthService {
	return NewHealthService(biz.NewIsBackendAvailableUsecase(d))
}

func createHealthService(spicedb *data.SpiceDbRepository) *HealthService {
	return NewHealthService(biz.NewIsBackendAvailableUsecase(spicedb))
}
