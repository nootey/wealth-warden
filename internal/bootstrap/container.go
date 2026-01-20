package bootstrap

import (
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/finance"
	"wealth-warden/pkg/mailer"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	Config             *config.Config
	DB                 *gorm.DB
	AuthzService       *authz.Service
	AuthService        *services.AuthService
	UserService        *services.UserService
	LoggingService     *services.LoggingService
	AccountService     *services.AccountService
	TransactionService *services.TransactionService
	SettingsService    *services.SettingsService
	ChartingService    *services.ChartingService
	RoleService        *services.RolePermissionService
	StatsService       *services.StatisticsService
	ImportService      *services.ImportService
	ExportService      *services.ExportService
	InvestmentService  *services.InvestmentService
	NotesService       *services.NotesService
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {

	// Initialize mailer
	mail := mailer.NewMailer(cfg, &mailer.MailConfig{From: cfg.Mailer.Username, FromName: "Wealth Warden Support"})

	// Initialize job queue system (In-Memory)
	// Can later be swapped to Redis/Kafka with zero change to service layer
	jobQueue := jobqueue.NewJobQueue(1, 25)
	jobDispatcher := &jobqueue.InMemoryDispatcher{Queue: jobQueue}
	authzSvc := authz.NewService(db, 5*time.Minute)

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRolePermissionRepositoryRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	chartingRepo := repositories.NewChartingRepository(db)
	statsRepo := repositories.NewStatisticsRepository(db)
	importRepo := repositories.NewImportRepository(db)
	exportRepo := repositories.NewExportRepository(db)
	investmentRepo := repositories.NewInvestmentRepository(db)
	notesRepo := repositories.NewNotesRepository(db)

	// Initialize price fetch client
	priceFetchClient, err := finance.NewPriceFetchClient(cfg.FinanceAPIBaseURL)
	if err != nil {
		logger.Warn("Failed to create price fetch client", zap.Error(err))
	}

	// Initialize currency converter
	currencyConverter := finance.NewCurrencyManager(priceFetchClient, investmentRepo)

	// Initialize services
	loggingService := services.NewLoggingService(loggingRepo)
	authService := services.NewAuthService(userRepo, roleRepo, settingsRepo, loggingRepo, jobDispatcher, mail)
	roleService := services.NewRolePermissionService(roleRepo, loggingRepo, jobDispatcher)
	userService := services.NewUserService(userRepo, roleRepo, loggingRepo, jobDispatcher, mail)
	accountService := services.NewAccountService(accountRepo, transactionRepo, settingsRepo, loggingRepo, jobDispatcher, currencyConverter)
	transactionService := services.NewTransactionService(transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher, currencyConverter)
	settingsService := services.NewSettingsService(cfg, logger.Named("settings_serv"), settingsRepo, userRepo, loggingRepo, jobDispatcher)
	chartingService := services.NewChartingService(chartingRepo, accountRepo, transactionRepo, statsRepo)
	statsService := services.NewStatisticsService(statsRepo, accountRepo, transactionRepo, settingsRepo)
	importService := services.NewImportService(importRepo, transactionRepo, accountRepo, investmentRepo, settingsRepo, loggingRepo, jobDispatcher)
	exportService := services.NewExportService(exportRepo, transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)
	investmentService := services.NewInvestmentService(investmentRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher, priceFetchClient, currencyConverter)
	notesService := services.NewNotesService(notesRepo, loggingRepo, jobDispatcher)

	return &Container{
		Config:             cfg,
		DB:                 db,
		AuthzService:       authzSvc,
		AuthService:        authService,
		UserService:        userService,
		LoggingService:     loggingService,
		AccountService:     accountService,
		TransactionService: transactionService,
		SettingsService:    settingsService,
		ChartingService:    chartingService,
		RoleService:        roleService,
		StatsService:       statsService,
		ImportService:      importService,
		ExportService:      exportService,
		InvestmentService:  investmentService,
		NotesService:       notesService,
	}, nil
}
