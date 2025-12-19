package tests

import (
	"context"
	"path/filepath"
	"time"
	"wealth-warden/internal/bootstrap"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database/seeders"

	"github.com/pressly/goose/v3"
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
	App       *bootstrap.Container
}

type ServiceIntegrationSuite struct {
	suite.Suite
	TC  *TestContainer
	Ctx context.Context
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

	migrationsPath := filepath.Join("..", "..", "storage", "migrations")
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

	// Build application container
	appContainer, err := bootstrap.NewContainer(cfg, db, l)
	s.Require().NoError(err, "failed to bootstrap app container")

	s.TC = &TestContainer{
		container: container,
		DB:        db,
		App:       appContainer,
	}
}

// SetupTest empties mutable tables between tests, ensuring a clean slate
func (s *ServiceIntegrationSuite) SetupTest() {

	const truncateTestTablesSQL = `
TRUNCATE TABLE
    transactions,
    transfers,
    balances,
    accounts,
    account_daily_snapshots
RESTART IDENTITY CASCADE;
`

	err := s.TC.DB.Exec(truncateTestTablesSQL).Error
	s.Require().NoError(err, "failed to truncate test tables")
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
}
