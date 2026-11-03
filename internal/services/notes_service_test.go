//go:build integration

package services_test

import (
	"fmt"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/suite"
)

type NotesServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationNotesServiceSuite(t *testing.T) {
	suite.Run(t, new(NotesServiceTestSuite))
}

func (s *NotesServiceTestSuite) TestFetchNotesPaginated_CursorWalk() {
	svc := s.TC.App.NotesService
	userID := int64(1)

	for i := range 7 {
		id, err := svc.InsertNote(s.Ctx, userID, &models.NoteReq{Content: fmt.Sprintf("Walk note %d", i)})
		s.Require().NoError(err)
		if i%3 == 0 {
			s.Require().NoError(svc.ToggleResolveState(s.Ctx, userID, id))
		}
	}

	tests.AssertCursorWalk(&s.ServiceIntegrationSuite, models.NoteSort, utils.PaginationParams{},
		func(p utils.PaginationParams) ([]models.Note, *utils.Paginator, error) {
			return svc.FetchNotesPaginated(s.Ctx, userID, p)
		})
}
