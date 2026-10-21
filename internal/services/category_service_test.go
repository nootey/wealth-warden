package services_test

import (
	"net/http"
	"slices"
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

func (s *CategoryServiceTestSuite) TestCategoryNecessity_InsertAndUpdate() {
	svc := s.TC.App.TransactionService
	user := s.newTestUser("necessity@test.local")
	need, want := models.CategoryNecessityNeed, models.CategoryNecessityWant

	load := func(id int64) models.Category {
		var c models.Category
		s.Require().NoError(s.TC.DB.WithContext(s.Ctx).First(&c, id).Error)
		return c
	}

	defaultID, err := svc.InsertCategory(s.Ctx, user.ID, &models.CategoryReq{DisplayName: "Hobbies", Classification: "expense"})
	s.Require().NoError(err)
	s.Equal(&want, load(defaultID).Necessity)

	needID, err := svc.InsertCategory(s.Ctx, user.ID, &models.CategoryReq{DisplayName: "Groceries", Classification: "expense", Necessity: &need})
	s.Require().NoError(err)
	s.Equal(&need, load(needID).Necessity)

	incomeID, err := svc.InsertCategory(s.Ctx, user.ID, &models.CategoryReq{DisplayName: "Tips", Classification: "income", Necessity: &need})
	s.Require().NoError(err)
	s.Nil(load(incomeID).Necessity)

	_, err = svc.UpdateCategory(s.Ctx, user.ID, needID, &models.CategoryReq{DisplayName: "Groceries", Classification: "expense"})
	s.Require().NoError(err)
	s.Equal(&need, load(needID).Necessity)

	_, err = svc.UpdateCategory(s.Ctx, user.ID, needID, &models.CategoryReq{DisplayName: "Groceries", Classification: "income", Necessity: &need})
	s.Require().NoError(err)
	s.Nil(load(needID).Necessity)

	_, err = svc.UpdateCategory(s.Ctx, user.ID, needID, &models.CategoryReq{DisplayName: "Groceries", Classification: "expense"})
	s.Require().NoError(err)
	s.Equal(&want, load(needID).Necessity)
}

func (s *CategoryServiceTestSuite) TestSeedDefaultCategories_SetsNecessity() {
	svc := s.TC.App.TransactionService
	user := s.newTestUser("seed-necessity@test.local")

	_, err := svc.SeedDefaultCategories(s.Ctx, user.ID)
	s.Require().NoError(err)

	need, want := models.CategoryNecessityNeed, models.CategoryNecessityWant
	expenseDef, ok := models.FindDefaultCategoryDef("expense")
	s.Require().True(ok)
	s.Require().NotEmpty(expenseDef.Needs)

	var cats []models.Category
	s.Require().NoError(s.TC.DB.WithContext(s.Ctx).Where("user_id = ?", user.ID).Find(&cats).Error)
	s.Require().NotEmpty(cats)

	needs := 0
	for _, c := range cats {
		switch {
		case c.Classification != "expense":
			s.Nil(c.Necessity, c.Name)
		case slices.Contains(expenseDef.Needs, c.DisplayName):
			s.Equal(&need, c.Necessity, c.Name)
			needs++
		default:
			s.Equal(&want, c.Necessity, c.Name)
		}
	}
	s.Equal(len(expenseDef.Needs), needs)
}
