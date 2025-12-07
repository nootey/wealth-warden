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

	migrationsPath := filepath.Join("..", "..", "pkg", "database", "migrations")
	err = goose.Up(sqlDB, migrationsPath)
	s.Require().NoError(err, "migrations failed")

	// Load test config
	configPath := filepath.Join("..", "..", "pkg", "config")
	cfg, err := config.LoadConfig(&configPath, "test")
	s.Require().NoError(err, "failed to load test configuration")

	logger := zap.NewNop() // Silent logger for tests

	// Seed basic data
	err = seeders.SeedDatabase(s.Ctx, db, logger, cfg, "basic")
	s.Require().NoError(err, "seeding failed")

	// Build application container
	appContainer, err := bootstrap.NewContainer(cfg, db, logger)
	s.Require().NoError(err, "failed to bootstrap app container")

	s.TC = &TestContainer{
		container: container,
		DB:        db,
		App:       appContainer,
	}
}

// SetupTest runs before each test - clean transactional tables
func (s *ServiceIntegrationSuite) SetupTest() {

	// Clean tables that change during tests
	tables := []string{
		"transfers",
		"transactions",
		"balances",
		"accounts",
	}

	for _, table := range tables {
		err := s.TC.DB.Exec("DELETE FROM " + table).Error
		s.Require().NoError(err, "failed to clean table: "+table)
	}
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

//func SetupTestContainer(t *testing.T, ctx context.Context) *TestContainer {
//	t.Helper()
//
//	// Start PostgreSQL container
//	container, err := postgres.Run(ctx,
//		"postgres:16-alpine",
//		postgres.WithDatabase("testdb"),
//		postgres.WithUsername("testuser"),
//		postgres.WithPassword("testpass"),
//		testcontainers.WithWaitStrategy(
//			wait.ForLog("database system is ready to accept connections").
//				WithOccurrence(2).
//				WithStartupTimeout(30*time.Second),
//		),
//	)
//	if err != nil {
//		t.Fatalf("failed to start container: %s", err)
//	}
//
//	// Connect GORM to the container DB
//	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
//	if err != nil {
//		t.Fatalf("failed to get connection: %s", err)
//	}
//
//	db, err := gorm.Open(postgresdriver.Open(connStr), &gorm.Config{
//		Logger: logger.Default.LogMode(logger.Silent),
//	})
//	if err != nil {
//		t.Fatalf("failed to connect: %s", err)
//	}
//
//	// Run migrations
//	sqlDB, err := db.DB()
//	if err != nil {
//		t.Fatalf("failed to get sql.DB: %s", err)
//	}
//
//	if err := goose.SetDialect("postgres"); err != nil {
//		t.Fatalf("goose dialect: %s", err)
//	}
//
//	migrationsPath := filepath.Join("..", "..", "pkg", "database", "migrations")
//	if err := goose.Up(sqlDB, migrationsPath); err != nil {
//		t.Fatalf("migrations failed: %s", err)
//	}
//
//	// Load test config
//	configPath := filepath.Join("..", "..", "pkg", "config")
//	cfg, err := config.LoadConfig(&configPath, "test")
//	if err != nil {
//		t.Fatalf("failed to load test configuration: %s", err)
//	}
//
//	// Seed basic data into the container DB
//	if err := seeders.SeedDatabase(ctx, db, nil, cfg, "basic"); err != nil {
//		t.Fatalf("seeding failed: %s", err)
//	}
//
//	// Build application container on top of this DB
//	logger := zap.NewNop()
//	appContainer, err := bootstrap.NewContainer(cfg, db, logger)
//	if err != nil {
//		t.Fatalf("failed to bootstrap app container: %s", err)
//	}
//
//	return &TestContainer{
//		container: container,
//		DB:        db,
//		App:       appContainer,
//	}
//}
//
//func (tc *TestContainer) Close(t *testing.T, ctx context.Context) {
//	t.Helper()
//
//	if tc.DB != nil {
//		if sqlDB, err := tc.DB.DB(); err == nil && sqlDB != nil {
//			_ = sqlDB.Close()
//		}
//	}
//
//	if tc.container != nil {
//		if err := tc.container.Terminate(ctx); err != nil {
//			t.Logf("container cleanup warning: %s", err)
//		}
//	}
//}
