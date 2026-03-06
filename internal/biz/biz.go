package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewCreateRelationshipsUsecase, NewReadRelationshipsUsecase, NewDeleteRelationshipsUsecase, NewCheckUsecase, NewCheckForUpdateUsecase, NewCheckBulkForUpdateUsecase, NewGetSubjectsUseCase, NewGetResourcesUseCase, NewIsBackendAvailableUsecase, NewImportBulkTuplesUsecase, NewAcquireLockUsecase, NewCheckBulkUsecase)
