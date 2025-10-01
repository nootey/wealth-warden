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
	chartingService := services.NewChartingService(cfg, ctx, chartingRepo, accountRepo, transactionRepo)
	statsService := services.NewStatisticsService(cfg, ctx, statsRepo, accountRepo)

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
	}, nil
}
