package services

import (
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/repositories"
)

type NotesServiceInterface interface {
}

type NotesService struct {
	repo          repositories.NotesRepositoryInterface
	loggingRepo   repositories.LoggingRepositoryInterface
	jobDispatcher jobqueue.JobDispatcher
}

func NewNotesService(
	repo *repositories.NotesRepository,
	loggingRepo *repositories.LoggingRepository,
	jobDispatcher jobqueue.JobDispatcher,
) *NotesService {
	return &NotesService{
		repo:          repo,
		loggingRepo:   loggingRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ NotesServiceInterface = (*NotesService)(nil)
