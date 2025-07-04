package services

import (
	"go.uber.org/zap"
	"wealth-warden/internal/jobs"
)

type DefaultServiceContext struct {
	Logger         *zap.Logger
	LoggingService *LoggingService
	AuthService    *AuthService
	JobDispatcher  jobs.JobDispatcher
}

func (c *DefaultServiceContext) Logging() *LoggingService {
	return c.LoggingService
}

func (c *DefaultServiceContext) Auth() *AuthService {
	return c.AuthService
}
