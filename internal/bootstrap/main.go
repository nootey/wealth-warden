package bootstrap

import (
	"gorm.io/gorm"
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
}

func NewContainer(cfg *config.Config, db *gorm.DB) *Container {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg)

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)
	recActionRepo := repositories.NewReoccurringActionsRepository(db)
	inflowRepo := repositories.NewInflowRepository(db)
	outflowRepo := repositories.NewOutflowRepository(db)
	budgetRepo := repositories.NewBudgetRepository(db)
	savingsRepo := repositories.NewSavingsRepository(db)

	// Initialize services
	budgetInterface := shared.NewBudgetInterface(budgetRepo, inflowRepo, outflowRepo)
	loggingService := services.NewLoggingService(cfg, loggingRepo)
	authService := services.NewAuthService(cfg, userRepo, loggingService, webClientMiddleware)
	userService := services.NewUserService(cfg, userRepo)
	recActionService := services.NewReoccurringActionService(recActionRepo, authService, loggingService)
	inflowService := services.NewInflowService(cfg, authService, loggingService, recActionService, budgetInterface, inflowRepo)
	outflowService := services.NewOutflowService(cfg, authService, loggingService, recActionService, budgetInterface, outflowRepo)
	budgetService := services.NewBudgetService(cfg, authService, loggingService, budgetInterface, budgetRepo)
	savingsService := services.NewSavingsService(cfg, authService, loggingService, recActionService, budgetInterface, savingsRepo)

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
	}
}
