package services

import (
	"go.uber.org/zap"
	"wealth-warden/internal/jobs"
	"wealth-warden/internal/repositories"
)

type DefaultServiceContext struct {
	Logger         *zap.Logger
	LoggingService *LoggingService
	AuthService    *AuthService
	JobDispatcher  jobs.JobDispatcher
	SettingsRepo   *repositories.SettingsRepository
}
