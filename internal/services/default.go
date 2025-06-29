package services

import "go.uber.org/zap"

type DefaultServiceContext struct {
	Logger         *zap.Logger
	LoggingService *LoggingService
	AuthService    *AuthService
}

func (c *DefaultServiceContext) Logging() *LoggingService {
	return c.LoggingService
}

func (c *DefaultServiceContext) Auth() *AuthService {
	return c.AuthService
}
