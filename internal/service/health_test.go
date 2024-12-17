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
	assert.Equal(t, &pb.GetLivezResponse{Status: "OK", Code: 200}, resp)
}

func TestHealthService_GetReadyz_SpiceDBAvailable(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	spicedb, err := container.CreateSpiceDbRepository()
	assert.NoError(t, err)

	service := createHealthService(spicedb)
	resp, err := service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &pb.GetReadyzResponse{Status: "OK", Code: 200}, resp)
}

func TestHealthService_GetReadyz_SpiceDBUnavailable(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	d := &DummyZanzibar{}
	service := createDummyHealthService(d)
	d.SetAvailable(false)
	resp, err := service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &pb.GetReadyzResponse{Status: "Unavailable", Code: 503}, resp)
}

func TestHealthService_GetReadyz_StillReadyAfterBackendLaterUnavailable(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	d := &DummyZanzibar{}
	service := createDummyHealthService(d)
	resp, err := service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &pb.GetReadyzResponse{Status: "Unavailable", Code: 503}, resp)

	d.SetAvailable(true)
	resp, err = service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &pb.GetReadyzResponse{Status: "OK", Code: 200}, resp)

	d.SetAvailable(false)
	resp, err = service.GetReadyz(ctx, &pb.GetReadyzRequest{})

	assert.NoError(t, err)
	assert.Equal(t, &pb.GetReadyzResponse{Status: "OK", Code: 200}, resp)
}

type DummyZanzibar struct {
	biz.ZanzibarRepository
	available bool
}

func (dz *DummyZanzibar) SetAvailable(available bool) {
	dz.available = available
}

func (dz *DummyZanzibar) IsBackendAvailable() error {
	if !dz.available {
		return fmt.Errorf("Unavailable")
	} else {
		return nil
	}
}

func createDummyHealthService(d *DummyZanzibar) *HealthService {
	return NewHealthService(biz.NewIsBackendAvailableUsecase(d))
}

func createHealthService(spicedb *data.SpiceDbRepository) *HealthService {
	return NewHealthService(biz.NewIsBackendAvailableUsecase(spicedb))
}
