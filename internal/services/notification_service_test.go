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

type NotificationServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestIntegrationNotificationServiceSuite(t *testing.T) {
	suite.Run(t, new(NotificationServiceTestSuite))
}

func (s *NotificationServiceTestSuite) TestGetNotifications_CursorWalk() {
	svc := s.TC.App.NotificationService
	userID := int64(1)

	for i := range 7 {
		s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Create(&models.Notification{
			UserID:  userID,
			Title:   fmt.Sprintf("Walk notification %d", i),
			Message: "Walk",
			Type:    models.NotificationTypeInfo,
		}).Error)
	}

	tests.AssertCursorWalk(&s.ServiceIntegrationSuite, models.NotificationSort, utils.PaginationParams{},
		func(p utils.PaginationParams) ([]models.Notification, *utils.Paginator, error) {
			return svc.GetNotifications(s.Ctx, userID, false, p)
		})
}
