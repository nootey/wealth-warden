package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/Finnhub-Stock-API/finnhub-go"
	"github.com/shopspring/decimal"
)

type InvestmentServiceInterface interface {
	FetchInvestmentHoldingsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentHolding, *utils.Paginator, error)
	FetchInvestmentTransactionsPaginated(ctx context.Context, userID int64, p utils.PaginationParams, accountID *int64) ([]models.InvestmentTransaction, *utils.Paginator, error)
	InsertHolding(ctx context.Context, userID int64, req *models.InvestmentHoldingReq) (int64, error)
}

type InvestmentService struct {
	repo          repositories.InvestmentRepositoryInterface
	accRepo       repositories.AccountRepositoryInterface
	settingsRepo  *repositories.SettingsRepository
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
	finnhubClient *finnhub.DefaultApiService
}

func NewInvestmentService(
	repo *repositories.InvestmentRepository,
	accRepo *repositories.AccountRepository,
	settingsRepo *repositories.SettingsRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
	finnhubClient *finnhub.DefaultApiService,
) *InvestmentService {
	return &InvestmentService{
		repo:          repo,
		accRepo:       accRepo,
		settingsRepo:  settingsRepo,
		jobDispatcher: jobDispatcher,
		loggingRepo:   loggingRepo,
		finnhubClient: finnhubClient,
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

	t := strings.ToUpper(req.Ticker)
	log.Printf("DEBUG: Investment type = '%s', Ticker = '%s'", req.InvestmentType, t)

	var ticker string
	switch req.InvestmentType {
	case models.InvestmentTypeCrypto:
		// Crypto format: "BINANCE:BTCUSDT" - use as-is
		ticker = t
	case models.InvestmentTypeStock, models.InvestmentTypeETF:
		// Stock/ETF format: "AAPL|L" -> "IWDA.L"
		if parts := strings.Split(t, "|"); len(parts) == 2 {
			ticker = parts[0] + "." + parts[1]
		} else {
			ticker = t // No exchange specified, try US (default)
		}
	}

	log.Printf("DEBUG2: Ticker = '%s'", ticker)

	quote, _, err := s.finnhubClient.Quote(ctx, ticker)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("error validating ticker %w", err)
	}

	if quote.C == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("invalid ticker '%s': no price data available", ticker)
	}

	currentPrice := decimal.NewFromFloat(float64(quote.C))
	now := time.Now()

	hold := models.InvestmentHolding{
		UserID:          userID,
		AccountID:       account.ID,
		InvestmentType:  req.InvestmentType,
		Name:            req.Name,
		Ticker:          req.Ticker,
		Quantity:        req.Quantity,
		AverageBuyPrice: decimal.Zero,
		CurrentPrice:    &currentPrice,
		LastPriceUpdate: &now,
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
