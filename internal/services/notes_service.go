package services

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"
)

type NotesServiceInterface interface {
	FetchNotesPaginated(ctx context.Context, userID int64, p utils.PaginationParams) ([]models.Note, *utils.Paginator, error)
	FetchNoteByID(ctx context.Context, userID int64, id int64) (*models.Note, error)
	InsertNote(ctx context.Context, userID int64, req *models.NoteReq) (int64, error)
	UpdateNote(ctx context.Context, userID, id int64, req *models.NoteReq) (int64, error)
	ToggleResolveState(ctx context.Context, userID int64, id int64) error
	DeleteNote(ctx context.Context, userID int64, id int64) error
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

func (s *NotesService) FetchNotesPaginated(ctx context.Context, userID int64, p utils.PaginationParams) ([]models.Note, *utils.Paginator, error) {
	totalRecords, err := s.repo.CountNotes(ctx, nil, userID)
	if err != nil {
		return nil, nil, err
	}

	offset := (p.PageNumber - 1) * p.RowsPerPage

	records, err := s.repo.FindNotes(ctx, nil, userID, offset, p.RowsPerPage)
	if err != nil {
		return nil, nil, err
	}

	from := offset + 1
	if from > int(totalRecords) {
		from = int(totalRecords)
	}

	to := offset + len(records)
	if to > int(totalRecords) {
		to = int(totalRecords)
	}

	paginator := &utils.Paginator{
		CurrentPage:  p.PageNumber,
		RowsPerPage:  p.RowsPerPage,
		TotalRecords: int(totalRecords),
		From:         from,
		To:           to,
	}

	return records, paginator, nil
}

func (s *NotesService) FetchNoteByID(ctx context.Context, userID int64, id int64) (*models.Note, error) {
	record, err := s.repo.FindNoteByID(ctx, nil, id, userID)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *NotesService) InsertNote(ctx context.Context, userID int64, req *models.NoteReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	note := models.Note{
		UserID:  userID,
		Content: req.Content,
	}

	noteID, err := s.repo.InsertNote(ctx, tx, &note)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(noteID, 10), changes, "id")
	utils.CompareChanges("", req.Content, changes, "content")

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "create",
		Category:    "note",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return noteID, nil
}

func (s *NotesService) UpdateNote(ctx context.Context, userID, id int64, req *models.NoteReq) (int64, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load existing note
	exNote, err := s.repo.FindNoteByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("can't find note with given id: %w", err)
	}

	note := models.Note{
		ID:      exNote.ID,
		UserID:  userID,
		Content: req.Content,
	}

	noteID, err := s.repo.UpdateNote(ctx, tx, note)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(exNote.Content, req.Content, changes, "content")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "note",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return noteID, nil
}

func (s *NotesService) ToggleResolveState(ctx context.Context, userID int64, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Load record to confirm it exists
	exNote, err := s.repo.FindNoteByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find note with given id: %w", err)
	}

	var resolvedAt *time.Time
	if exNote.ResolvedAt == nil {
		now := time.Now().UTC()
		resolvedAt = &now
	}

	note := models.Note{
		ID:         exNote.ID,
		UserID:     userID,
		ResolvedAt: resolvedAt,
	}

	_, err = s.repo.ToggleResolveState(ctx, tx, note)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()

	var exResolvedStr, resolvedStr string
	if exNote.ResolvedAt != nil {
		exResolvedStr = exNote.ResolvedAt.UTC().Format(time.RFC3339)
	} else {
		exResolvedStr = ""
	}

	if resolvedAt != nil {
		resolvedStr = resolvedAt.UTC().Format(time.RFC3339)
	} else {
		resolvedStr = ""
	}

	utils.CompareChanges(exResolvedStr, resolvedStr, changes, "resolved_at")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
			LoggingRepo: s.loggingRepo,
			Event:       "update",
			Category:    "note",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *NotesService) DeleteNote(ctx context.Context, userID int64, id int64) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Confirm existence
	note, err := s.repo.FindNoteByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("can't find note with given id: %w", err)
	}

	err = s.repo.DeleteNote(ctx, tx, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(note.Content, "", changes, "content")

	err = s.jobDispatcher.Dispatch(&jobqueue.ActivityLogJob{
		LoggingRepo: s.loggingRepo,
		Event:       "delete",
		Category:    "note",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return err
	}

	return nil
}
