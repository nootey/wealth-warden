package services

import "wealth-warden/internal/repositories"

type ReoccurringActionService struct {
	ActionRepo *repositories.ReoccurringActionsRepository
}

func NewReoccurringActionService(repo *repositories.ReoccurringActionsRepository) *ReoccurringActionService {
	return &ReoccurringActionService{
		ActionRepo: repo,
	}
}
