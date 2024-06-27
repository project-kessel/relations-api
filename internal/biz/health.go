package biz

type IsBackendAvaliableUsecase struct {
	repo ZanzibarRepository
}

func NewIsBackendAvailableUsecase(repo ZanzibarRepository) *IsBackendAvaliableUsecase {
	return &IsBackendAvaliableUsecase{repo: repo}
}

func (rc *IsBackendAvaliableUsecase) IsBackendAvailable() error {
	return rc.repo.IsBackendAvailable()
}
