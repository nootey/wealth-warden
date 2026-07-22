package services_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/suite"
	"gorm.io/datatypes"
)

type LoggingServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestLoggingServiceSuite(t *testing.T) {
	suite.Run(t, new(LoggingServiceTestSuite))
}

// activity_logs is not truncated by the shared SetupTest, so clear it per test
func (s *LoggingServiceTestSuite) clearLogs() {
	s.Require().NoError(s.TC.DB.Exec("DELETE FROM activity_logs").Error)
}

// The basic seeder only creates the root user, so a second causer has to be made
// to satisfy the activity_logs -> users foreign key
func (s *LoggingServiceTestSuite) createUser(email string) int64 {
	user := models.User{
		Email:       email,
		Password:    "x",
		DisplayName: email,
		RoleID:      1,
	}
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Create(&user).Error)
	return user.ID
}

func (s *LoggingServiceTestSuite) pageParams(page, rows int) utils.PaginationParams {
	return utils.PaginationParams{PageNumber: page, RowsPerPage: rows, SortField: "created_at", SortOrder: "desc"}
}

func (s *LoggingServiceTestSuite) insertLog(event, category string, recordID int64, causerID int64) {
	metadata, err := json.Marshal(map[string]map[string]interface{}{
		"new": {"id": recordID},
	})
	s.Require().NoError(err)

	log := models.ActivityLog{
		Event:    event,
		Category: category,
		Metadata: datatypes.JSON(metadata),
		CauserID: &causerID,
	}
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Create(&log).Error)
}

// Two users edit the same record id; each trail must show only its own causer's rows
func (s *LoggingServiceTestSuite) TestFetchAuditTrail_ScopedToCauser() {
	s.clearLogs()
	svc := s.TC.App.LoggingService

	owner := int64(1)
	other := s.createUser("other-causer@example.com")

	s.insertLog("update", "account", 42, owner)
	s.insertLog("update", "account", 42, other)

	trail, _, err := svc.FetchAuditTrail(s.Ctx, "42", []string{"account"}, []string{"update"}, owner, s.pageParams(1, 10))
	s.Require().NoError(err)
	s.Require().Len(trail, 1, "trail should contain only the caller's own log")
	s.Assert().Equal(owner, *trail[0].CauserID)
}

// A balance log stamps its account id, so an account trail spanning both
// categories returns them against a single subject id
func (s *LoggingServiceTestSuite) TestFetchAuditTrail_SpansCategoriesOnSubjectID() {
	s.clearLogs()
	svc := s.TC.App.LoggingService

	owner := int64(1)
	s.insertLog("update", "account", 7, owner)
	s.insertLog("update", "balance", 7, owner)
	s.insertLog("update", "balance", 8, owner)

	trail, _, err := svc.FetchAuditTrail(s.Ctx, "7", []string{"account", "balance"}, []string{"update"}, owner, s.pageParams(1, 10))
	s.Require().NoError(err)
	s.Require().Len(trail, 2, "trail should span both categories for the subject account")

	categories := []string{trail[0].Category, trail[1].Category}
	s.Assert().ElementsMatch([]string{"account", "balance"}, categories)
}

// Re-runs the realignment migration's Up block against pre-fix rows. The statements are
// idempotent, so applying them a second time is safe and exercises the real SQL
func (s *LoggingServiceTestSuite) TestRealignMigration_BackfillsLegacyRows() {
	s.clearLogs()

	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	raw, err := os.ReadFile(filepath.Join(root, "storage", "migrations",
		"20260721120000_realign_activity_log_categories_and_events.sql"))
	s.Require().NoError(err)

	up := string(raw)
	up = up[strings.Index(up, "-- +goose StatementBegin")+len("-- +goose StatementBegin"):]
	up = up[:strings.Index(up, "-- +goose StatementEnd")]

	var rootEmail string
	s.Require().NoError(s.TC.DB.Raw("SELECT email FROM users WHERE id = 1").Scan(&rootEmail).Error)

	s.Require().NoError(s.TC.DB.Exec(`
		INSERT INTO activity_logs (event, category, metadata, causer_id) VALUES
		('create', 'investment', '{"new":{"id":"5"},"old":{}}', 1),
		('update', 'account',    '{"new":{"id":"1","is_active":"true"},"old":{"is_active":"false"}}', 1),
		('update', 'account',    '{"new":{"id":"2","is_active":"false"},"old":{"is_active":"true"}}', 1),
		('update', 'account',    '{"new":{"id":"3","name":"Renamed"},"old":{"name":"Old"}}', 1),
		('delete', 'user',       '{"new":{},"old":{"email":"` + rootEmail + `"}}', 1)
	`).Error)

	s.Require().NoError(s.TC.DB.Exec(up).Error)

	type row struct {
		Event    string
		Category string
	}
	var got []row
	s.Require().NoError(s.TC.DB.Raw(
		`SELECT event, category FROM activity_logs ORDER BY id`).Scan(&got).Error)

	s.Assert().Equal(row{"create", "investment_asset"}, got[0], "asset category renamed")
	s.Assert().Equal(row{"restore", "account"}, got[1], "is_active true => restore")
	s.Assert().Equal(row{"deactivate", "account"}, got[2], "is_active false => deactivate")
	s.Assert().Equal(row{"update", "account"}, got[3], "plain account update untouched")

	var userID string
	s.Require().NoError(s.TC.DB.Raw(
		`SELECT metadata->'new'->>'id' FROM activity_logs WHERE category = 'user'`).Scan(&userID).Error)
	s.Assert().Equal("1", userID, "user delete backfilled with the subject id")
}

// The trail is windowed in SQL, so the paginator must count the whole trail
// while each page returns only its slice
func (s *LoggingServiceTestSuite) TestFetchAuditTrail_Paginates() {
	s.clearLogs()
	svc := s.TC.App.LoggingService

	owner := int64(1)
	for i := 0; i < 7; i++ {
		s.insertLog("update", "account", 21, owner)
	}

	first, paginator, err := svc.FetchAuditTrail(s.Ctx, "21", []string{"account"}, []string{"update"}, owner, s.pageParams(1, 3))
	s.Require().NoError(err)
	s.Assert().Len(first, 3)
	s.Assert().Equal(7, paginator.TotalRecords)
	s.Assert().Equal(1, paginator.From)
	s.Assert().Equal(3, paginator.To)

	last, paginator, err := svc.FetchAuditTrail(s.Ctx, "21", []string{"account"}, []string{"update"}, owner, s.pageParams(3, 3))
	s.Require().NoError(err)
	s.Assert().Len(last, 1, "final page holds the remainder")
	s.Assert().Equal(7, paginator.From)
	s.Assert().Equal(7, paginator.To)
}

// Walking to a record only another user has touched must reveal nothing
func (s *LoggingServiceTestSuite) TestFetchAuditTrail_ForeignRecordReturnsEmpty() {
	s.clearLogs()
	svc := s.TC.App.LoggingService

	owner := int64(1)
	other := s.createUser("foreign-causer@example.com")

	s.insertLog("update", "account", 99, other)

	trail, _, err := svc.FetchAuditTrail(s.Ctx, "99", []string{"account"}, []string{"update"}, owner, s.pageParams(1, 10))
	s.Require().NoError(err)
	s.Assert().Empty(trail, "another user's record history must not be readable")
}
