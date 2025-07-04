package bootstrap

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
	"wealth-warden/internal/services/shared"
	"wealth-warden/pkg/config"
)

type Container struct {
	Config                   *config.Config
	DB                       *gorm.DB
	Middleware               *middleware.WebClientMiddleware
	AuthService              *services.AuthService
	UserService              *services.UserService
	InflowService            *services.InflowService
	OutflowService           *services.OutflowService
	LoggingService           *services.LoggingService
	ReoccurringActionService *services.ReoccurringActionService
	BudgetService            *services.BudgetService
	SavingsService           *services.SavingsService
	InvestmentsService       *services.InvestmentsService
}

func NewContainer(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Container {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg)

	// Initialize job queue system (In-Memory)
	// Can later be swapped to Redis/Kafka with zero change to service layer
	jobQueue := jobs.NewJobQueue(1, 25)
	jobDispatcher := &jobs.InMemoryDispatcher{Queue: jobQueue}

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	recActionRepo := repositories.NewReoccurringActionsRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)
	investmentsRepo := repositories.NewInvestmentsRepository(db)

	// Initialize services
	budgetInterface := shared.NewBudgetInterface(budgetRepo, inflowRepo, outflowRepo)
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, logger, userRepo, loggingService, webClientMiddleware)

	ctx := &services.DefaultServiceContext{
		LoggingService: loggingService,
		AuthService:    authService,
		Logger:         logger,
		JobDispatcher:  jobDispatcher,
	}

	userService := services.NewUserService(cfg, ctx, userRepo)
	recActionService := services.NewReoccurringActionService(cfg, ctx, recActionRepo)
	inflowService := services.NewInflowService(cfg, ctx, inflowRepo, recActionService, budgetInterface)
	outflowService := services.NewOutflowService(cfg, ctx, outflowRepo, recActionService, budgetInterface)
	budgetService := services.NewBudgetService(cfg, ctx, budgetRepo, budgetInterface)
	savingsService := services.NewSavingsService(cfg, ctx, savingsRepo, budgetInterface, recActionService)
	investmentsService := services.NewInvestmentsService(cfg, ctx, investmentsRepo, recActionService)

	return &Container{
		Config:                   cfg,
		DB:                       db,
		Middleware:               webClientMiddleware,
		AuthService:              authService,
		UserService:              userService,
		InflowService:            inflowService,
		OutflowService:           outflowService,
		LoggingService:           loggingService,
		ReoccurringActionService: recActionService,
		BudgetService:            budgetService,
		SavingsService:           savingsService,
		InvestmentsService:       investmentsService,
	}
}
