package services_test

import (
	"net/http"
	"testing"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/models"
	"wealth-warden/internal/tests"

	"github.com/stretchr/testify/suite"
)

type CategoryServiceTestSuite struct {
	tests.ServiceIntegrationSuite
}

func TestCategoryServiceSuite(t *testing.T) {
	suite.Run(t, new(CategoryServiceTestSuite))
}

// newTestUser inserts a fresh user reusing the seeded user's role, so tests that
// need a second (or isolated) account don't depend on the "full" seed profile.
func (s *CategoryServiceTestSuite) newTestUser(email string) models.User {
	var seedUser models.User
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).First(&seedUser, seedUserID).Error)

	u := models.User{
		Email:       email,
		Password:    seedUser.Password,
		DisplayName: email,
		RoleID:      seedUser.RoleID,
	}
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Create(&u).Error)
	return u
}

// Two users must be able to use the same category name + classification: uniqueness
// is scoped per user now, not global.
func (s *CategoryServiceTestSuite) TestInsertCategory_SameNameAcrossUsers() {
	svc := s.TC.App.TransactionService
	other := s.newTestUser("other-user@test.local")

	req := &models.CategoryReq{DisplayName: "Freelance Income", Classification: "income"}

	_, err := svc.InsertCategory(s.Ctx, seedUserID, req)
	s.Require().NoError(err)

	_, err = svc.InsertCategory(s.Ctx, other.ID, req)
	s.Require().NoError(err, "a second user must be able to use the same category name")
}

// The same user creating the same category twice must get a clear conflict, not a
// raw unclassified 500.
func (s *CategoryServiceTestSuite) TestInsertCategory_DuplicateForSameUser() {
	svc := s.TC.App.TransactionService
	user := s.newTestUser("duplicate-category@test.local")
	req := &models.CategoryReq{DisplayName: "Side Hustle Pay", Classification: "income"}

	_, err := svc.InsertCategory(s.Ctx, user.ID, req)
	s.Require().NoError(err)

	_, err = svc.InsertCategory(s.Ctx, user.ID, req)
	s.Require().Error(err)
	status, _ := apperr.Resolve(err)
	s.Equal(http.StatusConflict, status)
}

// Seeding must be safe to call more than once: the second call adds nothing new.
func (s *CategoryServiceTestSuite) TestSeedDefaultCategories_Idempotent() {
	svc := s.TC.App.TransactionService
	user := s.newTestUser("seed-defaults@test.local")

	created, err := svc.SeedDefaultCategories(s.Ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().Greater(created, 0)

	createdAgain, err := svc.SeedDefaultCategories(s.Ctx, user.ID)
	s.Require().NoError(err)
	s.Assert().Equal(0, createdAgain)
}

// Once categories are per-user, editing and merging a default category must work
// like any other category.
func (s *CategoryServiceTestSuite) TestDefaultCategory_EditAndMerge() {
	svc := s.TC.App.TransactionService
	user := s.newTestUser("edit-merge@test.local")

	_, err := svc.SeedDefaultCategories(s.Ctx, user.ID)
	s.Require().NoError(err)

	var rent, salary, bonus models.Category
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Where("user_id = ? AND name = ?", user.ID, "rent").First(&rent).Error)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Where("user_id = ? AND name = ?", user.ID, "salary").First(&salary).Error)
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).
		Where("user_id = ? AND name = ?", user.ID, "bonus").First(&bonus).Error)

	// Editing a default category's classification must be allowed now.
	_, err = svc.UpdateCategory(s.Ctx, user.ID, rent.ID, &models.CategoryReq{
		DisplayName:    rent.DisplayName,
		Classification: "adjustment",
	})
	s.Require().NoError(err)

	// Merging FROM a default category must be allowed now too (same classification,
	// untouched by the edit above).
	_, err = svc.MergeCategories(s.Ctx, user.ID, salary.ID, bonus.ID)
	s.Require().NoError(err)
}
