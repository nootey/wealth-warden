package bootstrap

import (
	"time"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/authz"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/mailer"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Container struct {
	Config             *config.Config
	DB                 *gorm.DB
	Middleware         *middleware.WebClientMiddleware
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
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg, logger)

	// Initialize mailer
	mail := mailer.NewMailer(cfg, &mailer.MailConfig{From: cfg.Mailer.Username, FromName: "Wealth Warden Support"})

	// Initialize job queue system (In-Memory)
	// Can later be swapped to Redis/Kafka with zero change to service layer
	jobQueue := jobs.NewJobQueue(1, 25)
	jobDispatcher := &jobs.InMemoryDispatcher{Queue: jobQueue}
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

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, userRepo, roleRepo, settingsRepo, loggingRepo, webClientMiddleware, jobDispatcher, mail)
	roleService := services.NewRolePermissionService(cfg, roleRepo, loggingRepo, jobDispatcher)
	userService := services.NewUserService(cfg, userRepo, roleRepo, loggingRepo, jobDispatcher, mail)
	accountService := services.NewAccountService(cfg, accountRepo, transactionRepo, settingsRepo, loggingRepo, jobDispatcher)
	transactionService := services.NewTransactionService(cfg, transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)
	settingsService := services.NewSettingsService(cfg, settingsRepo, userRepo, loggingRepo, jobDispatcher)
	chartingService := services.NewChartingService(cfg, chartingRepo, accountRepo, transactionRepo)
	statsService := services.NewStatisticsService(cfg, statsRepo, accountRepo, transactionRepo)
	importService := services.NewImportService(cfg, importRepo, transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)
	exportService := services.NewExportService(cfg, exportRepo, transactionRepo, accountRepo, settingsRepo, loggingRepo, jobDispatcher)

	return &Container{
		Config:             cfg,
		DB:                 db,
		Middleware:         webClientMiddleware,
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
	}, nil
}
