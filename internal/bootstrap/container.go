package bootstrap

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/pkg/config"
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
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) (*Container, error) {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg, logger)

	// Initialize job queue system (In-Memory)
	// Can later be swapped to Redis/Kafka with zero change to service layer
	jobQueue := jobs.NewJobQueue(1, 25)
	jobDispatcher := &jobs.InMemoryDispatcher{Queue: jobQueue}

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	accountRepo := repositories.NewAccountRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)

	// Initialize services
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, logger, userRepo, loggingService, webClientMiddleware, jobDispatcher)

	ctx := &services.DefaultServiceContext{
		LoggingService: loggingService,
		AuthService:    authService,
		Logger:         logger,
		JobDispatcher:  jobDispatcher,
	}

	userService := services.NewUserService(cfg, ctx, userRepo)
	accountService := services.NewAccountService(cfg, ctx, accountRepo)
	transactionService := services.NewTransactionService(cfg, ctx, transactionRepo, accountService)

	return &Container{
		Config:             cfg,
		DB:                 db,
		Middleware:         webClientMiddleware,
		AuthService:        authService,
		UserService:        userService,
		LoggingService:     loggingService,
		AccountService:     accountService,
		TransactionService: transactionService,
	}, nil
}
