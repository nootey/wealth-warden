package services

import (
	"context"
	"fmt"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/prices"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
)

type InvestmentServiceInterface interface {
	FetchInvestmentHoldingsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentHolding, *utils.Paginator, error)
	FetchAllInvestmentHoldings(ctx context.Context, userID int64) ([]models.InvestmentHolding, error)
	FetchInvestmentHoldingByID(ctx context.Context, userID int64, id int64) (*models.InvestmentHolding, error)
	FetchInvestmentTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTransaction, *utils.Paginator, error)
	FetchInvestmentTransactionByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTransaction, error)
	InsertHolding(ctx context.Context, userID int64, req *models.InvestmentHoldingReq) (int64, error)
}

type InvestmentService struct {
	repo             repositories.InvestmentRepositoryInterface
	accRepo          repositories.AccountRepositoryInterface
	settingsRepo     *repositories.SettingsRepository
	loggingRepo      repositories.LoggingRepositoryInterface
	jobDispatcher    jobqueue.JobDispatcher
	priceFetchClient prices.PriceFetcher
}

func NewInvestmentService(
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
	priceFetchClient prices.PriceFetcher,
) *InvestmentService {
	return &InvestmentService{
		repo:             repo,
		accRepo:          accRepo,
		settingsRepo:     settingsRepo,
		jobDispatcher:    jobDispatcher,
		loggingRepo:      loggingRepo,
		priceFetchClient: priceFetchClient,
	}
}

var _ InvestmentServiceInterface = (*InvestmentService)(nil)

func (s *InvestmentService) FetchInvestmentHoldingsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentHolding, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentHoldings(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentHoldings(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *InvestmentService) FetchAllInvestmentHoldings(ctx context.Context, userID int64) ([]models.InvestmentHolding, error) {
	return s.repo.FindAllInvestmentHoldings(ctx, nil, userID)
}

func (s *InvestmentService) FetchInvestmentHoldingByID(ctx context.Context, userID int64, id int64) (*models.InvestmentHolding, error) {

	record, err := s.repo.FindInvestmentHoldingByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) FetchInvestmentTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTransaction, *utils.Paginator, error) {

	totalRecords, err := s.repo.CountInvestmentTransactions(ctx, nil, userID, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindInvestmentTransactions(ctx, nil, userID, offset, p.RowsPerPage, p.SortField, p.SortOrder, p.Filters, accountID)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *InvestmentService) FetchInvestmentTransactionByID(ctx context.Context, userID int64, id int64) (*models.InvestmentTransaction, error) {

	record, err := s.repo.FindInvestmentTransactionByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *InvestmentService) InsertHolding(ctx context.Context, userID int64, req *models.InvestmentHoldingReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	account, err := s.accRepo.FindAccountByID(ctx, tx, req.AccountID, userID, false)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find account with given id %w", err)
	}

	// Validate ticker and fetch price
	var currentPrice *decimal.Decimal
	var lastPriceUpdate *time.Time

	if s.priceFetchClient != nil {
		var ticker, exchangeOrCurrency string
		ut := strings.ToUpper(req.Ticker)

		switch req.InvestmentType {
		case models.InvestmentCrypto:
			// "BTC-USD"
			if parts := strings.Split(ut, "-"); len(parts) == 2 {
				ticker = parts[0]
				exchangeOrCurrency = parts[1]
			} else {
				ticker = ut
			}
		case models.InvestmentStock, models.InvestmentETF:
			// "IWDA.L"
			if parts := strings.Split(ut, "."); len(parts) == 2 {
				ticker = parts[0]
				exchangeOrCurrency = parts[1]
			} else {
				ticker = ut
			}
		}

		// Fetch price
		priceData, err := s.priceFetchClient.GetAssetPrice(ctx, ticker, req.InvestmentType, exchangeOrCurrency)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to fetch price for ticker '%s': %w", ticker, err)
		}

		price := decimal.NewFromFloat(priceData.Price)
		now := time.Unix(priceData.LastUpdate, 0)
		currentPrice = &price
		lastPriceUpdate = &now

	} else {
		// Client not available - allow creation but without price
		currentPrice = nil
		lastPriceUpdate = nil
	}

	hold := models.InvestmentHolding{
		UserID:          userID,
		AccountID:       account.ID,
		InvestmentType:  req.InvestmentType,
		Name:            req.Name,
		Ticker:          req.Ticker,
		Quantity:        req.Quantity,
		AverageBuyPrice: decimal.Zero,
		CurrentPrice:    currentPrice,
		LastPriceUpdate: lastPriceUpdate,
	}

	holdID, err := s.repo.InsertHolding(ctx, tx, &hold)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	quantityString := hold.Quantity.StringFixed(2)

	utils.CompareChanges("", account.Name, changes, "account")
	utils.CompareChanges("", hold.Name, changes, "name")
	utils.CompareChanges("", hold.Ticker, changes, "ticker")
	utils.CompareChanges("", string(hold.InvestmentType), changes, "type")
	utils.CompareChanges("", quantityString, changes, "quantity")

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "investment",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return holdID, nil
}
