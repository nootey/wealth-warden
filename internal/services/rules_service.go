package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"wealth-warden/internal/apperr"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type RulesServiceInterface interface {
	FetchRules(ctx context.Context, userID int64) ([]models.Rule, error)
	FetchRuleByID(ctx context.Context, userID, id int64) (*models.Rule, error)
	InsertRule(ctx context.Context, userID int64, req *models.RuleReq) (int64, error)
	UpdateRule(ctx context.Context, userID, id int64, req *models.RuleReq) (int64, error)
	DeleteRule(ctx context.Context, userID, id int64) error
	DispatchApplyRules(ctx context.Context, userID int64) error
}

// One page of uncategorized transactions per read while scanning.
const applyRulesBatchSize = 500

type RulesService struct {
	repo          repositories.RulesRepositoryInterface
	txnRepo       repositories.TransactionRepositoryInterface
	jobDispatcher jobqueue.Dispatcher
}

func NewRulesService(
	repo *repositories.RulesRepository,
	txnRepo *repositories.TransactionRepository,
	jobDispatcher jobqueue.Dispatcher,
) *RulesService {
	return &RulesService{
		repo:          repo,
		txnRepo:       txnRepo,
		jobDispatcher: jobDispatcher,
	}
}

var _ RulesServiceInterface = (*RulesService)(nil)

func (s *RulesService) FetchRules(ctx context.Context, userID int64) ([]models.Rule, error) {
	return s.repo.FindRules(ctx, nil, userID, false)
}

func (s *RulesService) FetchRuleByID(ctx context.Context, userID, id int64) (*models.Rule, error) {
	record, err := s.repo.FindRuleByID(ctx, nil, id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.Wrap(apperr.NotFound, "Rule not found", err)
		}
		return nil, err
	}
	return &record, nil
}

func buildRuleConditions(reqs []models.RuleConditionReq, depth int) ([]models.RuleCondition, error) {
	var out []models.RuleCondition
	for i, c := range reqs {
		if c.IsGroup {
			if depth > 0 {
				return nil, apperr.New(apperr.Validation, "A group cannot hold another group")
			}
			if c.MatchType == "" {
				return nil, apperr.New(apperr.Validation, "A group needs a match type")
			}
			if len(c.Conditions) == 0 {
				return nil, apperr.New(apperr.Validation, "A group needs at least one condition")
			}
			children, err := buildRuleConditions(c.Conditions, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, models.RuleCondition{IsGroup: true, MatchType: c.MatchType, Position: i, Children: children})
			continue
		}
		if c.Field == "" || c.Operator == "" || c.Value == "" {
			return nil, apperr.New(apperr.Validation, "A condition needs a field, an operator and a value")
		}
		switch c.Field {
		case models.RuleFieldDescription:
			if c.Operator != models.RuleOpContains {
				return nil, apperr.New(apperr.Validation, "A description condition only supports the contains operator")
			}
		case models.RuleFieldAmount:
			if c.Operator == models.RuleOpContains {
				return nil, apperr.New(apperr.Validation, "An amount condition does not support the contains operator")
			}
			if _, err := decimal.NewFromString(c.Value); err != nil {
				return nil, apperr.New(apperr.Validation, fmt.Sprintf("An amount condition needs a numeric value, got %q", c.Value))
			}
		case models.RuleFieldDirection:
			if c.Operator != models.RuleOpEquals {
				return nil, apperr.New(apperr.Validation, "A direction condition only supports the equals operator")
			}
			if !models.TransactionDirection(c.Value).IsValid() {
				return nil, apperr.New(apperr.Validation, fmt.Sprintf("A direction condition needs income or expense, got %q", c.Value))
			}
		}
		out = append(out, models.RuleCondition{Field: c.Field, Operator: c.Operator, Value: c.Value, Position: i})
	}
	return out, nil
}

func (s *RulesService) buildRule(ctx context.Context, tx *gorm.DB, userID int64, req *models.RuleReq) (models.Rule, error) {
	rule := models.Rule{UserID: userID, Name: req.Name, IsActive: true}
	if req.IsActive != nil {
		rule.IsActive = *req.IsActive
	}

	conditions, err := buildRuleConditions(req.Conditions, 0)
	if err != nil {
		return rule, err
	}
	rule.MatchType = req.MatchType
	rule.Conditions = conditions

	for i, a := range req.Actions {
		categoryID, err := strconv.ParseInt(a.Value, 10, 64)
		if err != nil {
			return rule, apperr.New(apperr.Validation, "The set_category action needs a category id")
		}
		if _, err := s.txnRepo.FindCategoryByID(ctx, tx, categoryID, userID, false); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rule, ErrInvalidCategoryID
			}
			return rule, err
		}
		rule.Actions = append(rule.Actions, models.RuleAction{
			ActionType: a.ActionType,
			Value:      a.Value,
			Position:   i,
		})
	}

	return rule, nil
}

func (s *RulesService) InsertRule(ctx context.Context, userID int64, req *models.RuleReq) (int64, error) {
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

	rule, err := s.buildRule(ctx, tx, userID, req)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	ruleID, err := s.repo.InsertRule(ctx, tx, &rule)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges("", strconv.FormatInt(ruleID, 10), changes, "id")
	utils.CompareChanges("", req.Name, changes, "name")

	err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "create",
		Category:    "rule",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
	if err != nil {
		return 0, err
	}

	return ruleID, nil
}

func (s *RulesService) UpdateRule(ctx context.Context, userID, id int64, req *models.RuleReq) (int64, error) {
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

	exRule, err := s.repo.FindRuleByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apperr.Wrap(apperr.NotFound, "Rule not found", err)
		}
		return 0, err
	}

	rule, err := s.buildRule(ctx, tx, userID, req)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	rule.ID = exRule.ID

	if _, err := s.repo.UpdateRule(ctx, tx, rule); err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(exRule.Name, rule.Name, changes, "name")
	utils.CompareChanges(strconv.FormatBool(exRule.IsActive), strconv.FormatBool(rule.IsActive), changes, "is_active")

	if !changes.IsEmpty() {
		err = s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
			Event:       "update",
			Category:    "rule",
			Description: nil,
			Payload:     changes,
			Causer:      &userID,
		})
		if err != nil {
			return 0, err
		}
	}

	return rule.ID, nil
}

func (s *RulesService) DeleteRule(ctx context.Context, userID, id int64) error {
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

	rule, err := s.repo.FindRuleByID(ctx, tx, id, userID)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperr.Wrap(apperr.NotFound, "Rule not found", err)
		}
		return err
	}

	if err := s.repo.DeleteRule(ctx, tx, id); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	changes := utils.InitChanges()
	utils.CompareChanges(strconv.FormatInt(rule.ID, 10), "", changes, "id")
	utils.CompareChanges(rule.Name, "", changes, "name")

	return s.jobDispatcher.Dispatch(ctx, jobqueue.ActivityLogArgs{
		Event:       "delete",
		Category:    "rule",
		Description: nil,
		Payload:     changes,
		Causer:      &userID,
	})
}

func (s *RulesService) DispatchApplyRules(ctx context.Context, userID int64) error {
	return s.jobDispatcher.Dispatch(ctx, jobqueue.ApplyRulesArgs{UserID: userID})
}

// ApplyRules scans a user's uncategorized transactions and sets a category on
// each one an active rule matches. The first matching rule wins, same as import.
// Already categorized transactions are never touched. It returns how many rows
// it scanned and how many it recategorized.
func (s *RulesService) ApplyRules(ctx context.Context, userID int64) (scanned int, categorized int, err error) {
	rules, err := s.repo.FindRules(ctx, nil, userID, true)
	if err != nil {
		return 0, 0, err
	}
	if len(rules) == 0 {
		return 0, 0, nil
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	// Drop rules whose target category no longer exists, so a broken rule cannot
	// point transactions at a missing category. Import skips them the same way.
	validRules, err := s.filterRulesWithExistingCategory(ctx, tx, userID, rules)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	if len(validRules) == 0 {
		tx.Rollback()
		return 0, 0, nil
	}

	uncategorized, err := s.txnRepo.EnsureRootCategory(ctx, tx, "uncategorized", userID)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// Collect matches read-only, paged by id, then bulk update per target category.
	byCategory := map[int64][]int64{}
	var afterID int64
	for {
		batch, err := s.txnRepo.FindUncategorizedTransactions(ctx, tx, userID, uncategorized.ID, afterID, applyRulesBatchSize)
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
		for _, t := range batch {
			scanned++
			afterID = t.ID
			cats := utils.MatchingRuleCategories(validRules, utils.SafeString(t.Description), t.Amount, t.Direction)
			if len(cats) == 0 {
				continue
			}
			target := cats[0]
			if t.CategoryID != nil && *t.CategoryID == target {
				continue
			}
			byCategory[target] = append(byCategory[target], t.ID)
		}
		if len(batch) < applyRulesBatchSize {
			break
		}
	}

	for categoryID, ids := range byCategory {
		moved, err := s.txnRepo.BulkSetTransactionCategoryByIDs(ctx, tx, ids, categoryID, userID)
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
		categorized += int(moved)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, 0, err
	}
	return scanned, categorized, nil
}

func (s *RulesService) filterRulesWithExistingCategory(ctx context.Context, tx *gorm.DB, userID int64, rules []models.Rule) ([]models.Rule, error) {
	exists := map[int64]bool{}
	var valid []models.Rule
	for _, rule := range rules {
		categoryID, ok := rule.CategoryID()
		if !ok {
			continue
		}
		found, cached := exists[categoryID]
		if !cached {
			_, lookupErr := s.txnRepo.FindCategoryByID(ctx, tx, categoryID, userID, false)
			switch {
			case lookupErr == nil:
				found = true
			case errors.Is(lookupErr, gorm.ErrRecordNotFound):
				found = false
			default:
				return nil, lookupErr
			}
			exists[categoryID] = found
		}
		if found {
			valid = append(valid, rule)
		}
	}
	return valid, nil
}
