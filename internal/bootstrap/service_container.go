package bootstrap

import (
	"time"
	"wealth-warden/internal/queue"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/mailer"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ServiceContainer struct {
	Config             *config.Config
	DB                 *gorm.DB
	AuthzService       *authz.Service
	BackofficeService  *services.BackofficeService
	AuthService        *services.AuthService
	UserService        *services.UserService
	LoggingService     *services.LoggingService
	AccountService     *services.AccountService
	TransactionService *services.TransactionService
	SettingsService    *services.SettingsService
	RoleService        *services.RolePermissionService
	ImportService      *services.ImportService
	ExportService      *services.ExportService
	InvestmentService  *services.InvestmentService
	NotesService       *services.NotesService
	AnalyticsService   *services.AnalyticsService
	SavingsService     *services.SavingsService
}

// NewServiceContainer initialises the application service layer.
// Pass a non-nil priceFetcher to override the default client (e.g. a mock in tests).
// Pass nil to have the real Yahoo Finance client created from cfg.FinanceAPIBaseURL.
func NewServiceContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger, jobDispatcher queue.JobDispatcher, priceFetcher finance.PriceFetcher) (*ServiceContainer, error) {
	if priceFetcher == nil {
		var err error
		priceFetcher, err = finance.NewPriceFetchClient(cfg.FinanceAPIBaseURL)
		if err != nil {
			logger.Warn("Failed to create price fetch client", zap.Error(err))
		}
	}

	// Initialize mailer
	mail := mailer.NewMailer(cfg, &mailer.MailConfig{From: cfg.Mailer.Username, FromName: "Wealth Warden Support"})

	// Initialize permission gating
	authzSvc := authz.NewService(db, 5*time.Minute)

	// Initialize repositories
	backOfficeRepo := repositories.NewBackofficeRepository(db)
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRolePermissionRepositoryRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	importRepo := repositories.NewImportRepository(db)
	exportRepo := repositories.NewExportRepository(db)
	investmentRepo := repositories.NewInvestmentRepository(db)
	notesRepo := repositories.NewNotesRepository(db)
	analyticsRepo := repositories.NewAnalyticsRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(loggingRepo)
	authService := services.NewAuthService(userRepo, roleRepo, settingsRepo, loggingRepo, jobDispatcher, mail)
	roleService := services.NewRolePermissionService(roleRepo, loggingRepo, jobDispatcher)
	userService := services.NewUserService(userRepo, roleRepo, loggingRepo, jobDispatcher, mail)
	accountService := services.NewAccountService(logger.Named("account_srv"), accountRepo, transactionRepo, settingsRepo, loggingRepo, investmentRepo, jobDispatcher)
	transactionService := services.NewTransactionService(transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)
	settingsService := services.NewSettingsService(cfg, logger.Named("settings_srv"), settingsRepo, userRepo, loggingRepo, transactionRepo, jobDispatcher)
	importService := services.NewImportService(importRepo, transactionRepo, accountRepo, investmentRepo, settingsRepo, loggingRepo, jobDispatcher)
	exportService := services.NewExportService(exportRepo, transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)
	investmentService := services.NewInvestmentService(logger.Named("investment_sev"), investmentRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher, priceFetcher)
	notesService := services.NewNotesService(notesRepo, loggingRepo, jobDispatcher)
	analyticsService := services.NewAnalyticsService(analyticsRepo, accountRepo, transactionRepo, settingsRepo)
	backOfficeService := services.NewBackofficeService(logger.Named("backoffice_srv"), jobDispatcher, backOfficeRepo, investmentService, accountService, userService)
	savingsService := services.NewSavingsService(savingsRepo, loggingRepo, jobDispatcher)

	return &ServiceContainer{
		Config:             cfg,
		DB:                 db,
		BackofficeService:  backOfficeService,
		AuthzService:       authzSvc,
		AuthService:        authService,
		UserService:        userService,
		LoggingService:     loggingService,
		AccountService:     accountService,
		TransactionService: transactionService,
		SettingsService:    settingsService,
		RoleService:        roleService,
		ImportService:      importService,
		ExportService:      exportService,
		InvestmentService:  investmentService,
		NotesService:       notesService,
		AnalyticsService:   analyticsService,
		SavingsService:     savingsService,
	}, nil
}
