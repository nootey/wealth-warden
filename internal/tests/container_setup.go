package tests

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/internal/config"
	"wealth-warden/internal/jobqueue"
	"wealth-warden/pkg/database/seeders"

	"github.com/alicebob/miniredis/v2"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TestContainer struct {
	container *postgres.PostgresContainer
	DB        *gorm.DB
	App       *bootstrap.ServiceContainer
	// For code that bypasses GORM (River's pgx pool).
	DSN string
}

type ServiceIntegrationSuite struct {
	suite.Suite
	TC    *TestContainer
	Ctx   context.Context
	redis *miniredis.Miniredis
}

type NoOpDispatcher struct{}

func (d *NoOpDispatcher) Dispatch(_ context.Context, _ jobqueue.Job) error {
	return nil
}

// SetupSuite runs once before all integration tests
func (s *ServiceIntegrationSuite) SetupSuite() {
	s.Ctx = context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(s.Ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	s.Require().NoError(err, "failed to start container")

	// Connect GORM to the container DB
	connStr, err := container.ConnectionString(s.Ctx, "sslmode=disable")
	s.Require().NoError(err, "failed to get connection string")

	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	s.Require().NoError(err, "failed to connect to database")

	// Run migrations
	sqlDB, err := db.DB()
	s.Require().NoError(err, "failed to get sql.DB")

	err = goose.SetDialect("postgres")
	s.Require().NoError(err, "failed to set goose dialect")

	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	migrationsPath := filepath.Join(projectRoot, "storage", "migrations")
	err = goose.Up(sqlDB, migrationsPath)
	s.Require().NoError(err, "migrations failed")

	// Load test config
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		panic(err)
	}
	cfg.Postgres.Database = "wealth_warden_test"
	s.Require().NoError(err, "failed to load test configuration")

	l := zap.NewNop() // Silent logger for tests

	// Seed basic data
	err = seeders.SeedDatabase(s.Ctx, db, l, cfg, "basic")
	s.Require().NoError(err, "seeding failed")

	// In-memory redis for the session store
	mr := miniredis.NewMiniRedis()
	s.Require().NoError(mr.Start(), "failed to start miniredis")
	s.redis = mr
	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	// Build application container
	jobDispatcher := &NoOpDispatcher{}
	appContainer, err := bootstrap.NewServiceContainer(cfg, db, redisClient, l, jobDispatcher, jobqueue.NoopJobManager{}, &MockPriceFetcher{})
	s.Require().NoError(err, "failed to bootstrap app container")

	s.TC = &TestContainer{
		container: container,
		DB:        db,
		App:       appContainer,
		DSN:       connStr,
	}
}

const truncateTestTablesSQL = `
TRUNCATE TABLE
    transactions,
    transfers,
    balances,
    accounts,
    balance_snapshots,
    ticker_price_history,
    rules,
    rule_conditions,
    rule_actions
RESTART IDENTITY CASCADE;
`

// TruncateMutableTables resets every table a test may write to. CASCADE also
// clears investment_assets and investment_trades, which hang off accounts.
func (s *ServiceIntegrationSuite) TruncateMutableTables() {
	err := s.TC.DB.Exec(truncateTestTablesSQL).Error
	s.Require().NoError(err, "failed to truncate test tables")
}

func (s *ServiceIntegrationSuite) LockAccount(accountID int64) *gorm.DB {
	tx := s.TC.DB.WithContext(s.Ctx).Begin()
	s.Require().NoError(tx.Error)
	// A failed test must not leave the lock held, or the next test's truncate waits on it forever
	s.T().Cleanup(func() { tx.Rollback() })

	var locked []int64
	s.Require().NoError(tx.Raw(`SELECT account_id FROM balances WHERE account_id = ? FOR UPDATE`, accountID).Scan(&locked).Error)
	s.Require().Len(locked, 1, "account %d has no balance row to lock", accountID)
	return tx
}

func (s *ServiceIntegrationSuite) WaitsFor(release func() error, call func() error) error {
	done := make(chan error, 1)
	go func() { done <- call() }()

	select {
	case err := <-done:
		s.Require().FailNowf("call did not wait for the lock", "returned %v", err)
	case <-time.After(300 * time.Millisecond):
	}

	s.Require().NoError(release())
	return <-done
}

func (s *ServiceIntegrationSuite) RunConcurrently(n int, call func(i int) error) []error {
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = call(i)
		}()
	}
	close(start)
	wg.Wait()
	return errs
}

// SetupTest empties mutable tables and the session store between tests, ensuring a clean slate
func (s *ServiceIntegrationSuite) SetupTest() {
	s.TruncateMutableTables()
	s.redis.FlushAll()
}

// TearDownSuite runs once after all tests
func (s *ServiceIntegrationSuite) TearDownSuite() {
	if s.TC.DB != nil {
		if sqlDB, err := s.TC.DB.DB(); err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	if s.TC.container != nil {
		if err := s.TC.container.Terminate(s.Ctx); err != nil {
			s.T().Logf("container cleanup warning: %s", err)
		}
	}

	if s.redis != nil {
		s.redis.Close()
	}
}
