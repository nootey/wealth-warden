package bootstrap

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/mailer"
)

type Container struct {
	Config             *config.Config
	DB                 *gorm.DB
	Middleware         *middleware.WebClientMiddleware
	AuthService        *services.AuthService
	UserService        *services.UserService
	LoggingService     *services.LoggingService
	AccountService     *services.AccountService
	TransactionService *services.TransactionService
	SettingsService    *services.SettingsService
	ChartingService    *services.ChartingService
	RoleService        *services.RolePermissionService
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

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRolePermissionRepositoryRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	settingsRepo := repositories.NewSettingsRepository(db)
	chartingRepo := repositories.NewChartingRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, logger, userRepo, roleRepo, settingsRepo, loggingService, webClientMiddleware, jobDispatcher, mail)

	ctx := &services.DefaultServiceContext{
		LoggingService: loggingService,
		AuthService:    authService,
		Logger:         logger,
		JobDispatcher:  jobDispatcher,
		SettingsRepo:   settingsRepo,
	}

	roleService := services.NewRolePermissionService(cfg, ctx, roleRepo)
	userService := services.NewUserService(cfg, ctx, userRepo, roleService)
	accountService := services.NewAccountService(cfg, ctx, accountRepo, transactionRepo)
	transactionService := services.NewTransactionService(cfg, ctx, transactionRepo, accountService)
	settingsService := services.NewSettingsService(cfg, ctx, settingsRepo)
	chartingService := services.NewChartingService(cfg, ctx, chartingRepo)

	return &Container{
		Config:             cfg,
		DB:                 db,
		Middleware:         webClientMiddleware,
		AuthService:        authService,
		UserService:        userService,
		LoggingService:     loggingService,
		AccountService:     accountService,
		TransactionService: transactionService,
		SettingsService:    settingsService,
		ChartingService:    chartingService,
		RoleService:        roleService,
	}, nil
}
