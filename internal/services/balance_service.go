package services

import (
	"context"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"

	"go.uber.org/zap"
)

type BalanceServiceInterface interface {
	ReconcileBalances(ctx context.Context) ([]models.BalanceDrift, error)
}

type BalanceService struct {
	repo   repositories.BalanceRepositoryInterface
	logger *zap.Logger
}

func NewBalanceService(
	logger *zap.Logger,
	repo *repositories.BalanceRepository,
) *BalanceService {
	return &BalanceService{
		repo:   repo,
		logger: logger,
	}
}

var _ BalanceServiceInterface = (*BalanceService)(nil)

func (s *BalanceService) ReconcileBalances(ctx context.Context) ([]models.BalanceDrift, error) {
	var repaired []models.BalanceDrift

	for afterID := int64(0); ; {
		ids, err := s.repo.FindOpenAccountIDs(ctx, nil, afterID, 500)
		if err != nil {
			return repaired, err
		}
		if len(ids) == 0 {
			return repaired, nil
		}
		afterID = ids[len(ids)-1]

		suspects, err := s.repo.FindDriftedAccounts(ctx, nil, ids)
		if err != nil {
			return repaired, err
		}

		for _, suspect := range suspects {
			drift, fixed, err := s.repairAccount(ctx, suspect.AccountID)
			if err != nil {
				return repaired, err
			}
			if fixed {
				repaired = append(repaired, drift)
			}
		}

		if ctx.Err() != nil {
			return repaired, ctx.Err()
		}
	}
}

func (s *BalanceService) repairAccount(ctx context.Context, accountID int64) (models.BalanceDrift, bool, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return models.BalanceDrift{}, false, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	drift, fixed, err := s.repo.RepairBalance(ctx, tx, accountID)
	if err != nil {
		tx.Rollback()
		return models.BalanceDrift{}, false, err
	}
	if err := tx.Commit().Error; err != nil {
		return models.BalanceDrift{}, false, err
	}
	return drift, fixed, nil
}
