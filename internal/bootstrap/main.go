package bootstrap

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/middleware"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/internal/services"
)

type Container struct {
	Config         *models.Config
	DB             *gorm.DB
	Middleware     *middleware.WebClientMiddleware
	AuthService    *services.AuthService
	UserService    *services.UserService
	LoggingService *services.LoggingService
}

func NewContainer(cfg *models.Config, db *gorm.DB, logger *zap.Logger) *Container {

	// Initialize middleware
	webClientMiddleware := middleware.NewWebClientMiddleware(cfg, logger)

	// Initialize job queue system (In-Memory)
	// Can later be swapped to Redis/Kafka with zero change to service layer
	jobQueue := jobs.NewJobQueue(1, 25)
	jobDispatcher := &jobs.InMemoryDispatcher{Queue: jobQueue}

	// Initialize repositories
	loggingRepo := repositories.NewLoggingRepository(db)
	userRepo := repositories.NewUserRepository(db)

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

	return &Container{
		Config:         cfg,
		DB:             db,
		Middleware:     webClientMiddleware,
		AuthService:    authService,
		UserService:    userService,
		LoggingService: loggingService,
	}
}
