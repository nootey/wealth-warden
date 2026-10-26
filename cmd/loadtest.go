//go:build loadtest

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"wealth-warden/internal/app"
	"wealth-warden/internal/config"
	"wealth-warden/pkg/database"
	"wealth-warden/pkg/database/seeders"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Fixed, so the k6 ledger check can reach the database with docker exec
const (
	loadtestContainer = "wealth-warden-loadtest-postgres"
	loadtestDatabase  = "wealth_warden_loadtest"
	loadtestUser      = "loadtest"
	loadtestPassword  = "loadtest"
	loadtestRandSeed  = 42
	loadtestLogFile   = "logs/loadtest.log"
	loadtestCheckSQL  = "scripts/k6/ledger_check.sql"
)

var loadtestUsers = map[string]int{
	"ledger":    1,
	"dashboard": 50,
	"stress":    1000,
}

var loadtestCmd = &cobra.Command{
	Use:          "loadtest [ledger|dashboard|stress]",
	Short:        "Run a k6 script against the app on throwaway containers with a lean bulk seed",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		test := args[0]
		users, ok := loadtestUsers[test]
		if !ok {
			return fmt.Errorf("unknown test %q, valid tests are: ledger, dashboard, stress", test)
		}

		port, err := cmd.Flags().GetString("port")
		if err != nil {
			return err
		}
		override, err := cmd.Flags().GetInt("users")
		if err != nil {
			return err
		}
		if override > 0 {
			users = override
		}
		serve, err := cmd.Flags().GetBool("serve")
		if err != nil {
			return err
		}

		if !serve {
			if _, err := exec.LookPath("k6"); err != nil {
				return fmt.Errorf("k6 is not installed: %w", err)
			}
		}

		// The terminal belongs to k6: app logs and per-request lines would bury its
		// output, and printing every request costs CPU under load
		logFile, err := os.Create(loadtestLogFile)
		if err != nil {
			return fmt.Errorf("failed to create the log file: %w", err)
		}
		encoder := zap.NewProductionEncoderConfig()
		encoder.EncodeTime = zapcore.ISO8601TimeEncoder
		logger = zap.New(
			zapcore.NewCore(zapcore.NewJSONEncoder(encoder), zapcore.AddSync(logFile), zapcore.InfoLevel),
			zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel),
		)
		log.SetOutput(logFile)
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = logFile

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()

		fmt.Println("Starting Postgres and Redis ...")

		pg, err := postgres.Run(ctx, "postgres:18",
			postgres.WithDatabase(loadtestDatabase),
			postgres.WithUsername(loadtestUser),
			postgres.WithPassword(loadtestPassword),
			testcontainers.WithName(loadtestContainer),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(30*time.Second),
			),
		)
		defer func() { _ = testcontainers.TerminateContainer(pg) }()
		if err != nil {
			return fmt.Errorf("failed to start postgres container (is another load test running?): %w", err)
		}

		rd, err := testcontainers.Run(ctx, "redis:7-alpine",
			testcontainers.WithExposedPorts("6379/tcp"),
			testcontainers.WithWaitStrategy(wait.ForListeningPort("6379/tcp")),
		)
		defer func() { _ = testcontainers.TerminateContainer(rd) }()
		if err != nil {
			return fmt.Errorf("failed to start redis container: %w", err)
		}

		pgHost, err := pg.Host(ctx)
		if err != nil {
			return err
		}
		pgPort, err := pg.MappedPort(ctx, "5432/tcp")
		if err != nil {
			return err
		}
		rdHost, err := rd.Host(ctx)
		if err != nil {
			return err
		}
		rdPort, err := rd.MappedPort(ctx, "6379/tcp")
		if err != nil {
			return err
		}

		cfg.Postgres.Host = pgHost
		cfg.Postgres.Port = pgPort.Int()
		cfg.Postgres.Database = loadtestDatabase
		cfg.Postgres.User = loadtestUser
		cfg.Postgres.Password = loadtestPassword
		cfg.Redis.Host = rdHost
		cfg.Redis.Port = rdPort.Int()
		cfg.Redis.Password = ""
		cfg.Redis.DB = 0
		cfg.HttpServer.Port = port

		// Pinned, not read from the dev config, so every run seeds the same data.
		// Domain and password match the k6 script defaults.
		cfg.Seed.Bulk = config.BulkSeedConfig{
			Users:       users,
			Years:       2,
			TxnsPerUser: 500,
			EmailDomain: "seed.local",
			Password:    "password",
			Lean:        true,
			RandSeed:    loadtestRandSeed,
		}

		fmt.Printf("Migrating and seeding %d lean bulk users ...\n", users)
		if err := prepareLoadtestDatabase(ctx, cfg, logger.Named("loadtest")); err != nil {
			return err
		}

		a, err := app.New(cfg, logger)
		if err != nil {
			return err
		}

		appCtx, stopApp := context.WithCancel(ctx)
		var appErr error
		appDone := make(chan struct{})
		go func() {
			appErr = a.Run(appCtx)
			close(appDone)
		}()
		// Runs before the container defers, so the app never sees its database vanish
		defer func() {
			fmt.Println("\nStopping the app and removing the containers ...")
			stopApp()
			<-appDone
		}()

		baseURL := "http://localhost:" + port
		if err := waitForLoadtestServer(ctx, baseURL, appDone, &appErr); err != nil {
			return err
		}

		env := loadtestK6Env(test, baseURL, users)

		if serve {
			fmt.Printf("\nServer on %s with %d bulk users, logs in %s. Run k6 with:\n\n  %s k6 run scripts/k6/%s.js\n\nPress Ctrl+C to stop the server and remove the containers.\n",
				baseURL, users, loadtestLogFile, strings.Join(env, " "), test)
			<-appDone
			return appErr
		}

		fmt.Printf("Server on %s, logs in %s. Running k6 ...\n\n", baseURL, loadtestLogFile)
		k6Err := runK6(test, env)
		if ctx.Err() != nil {
			return errors.New("interrupted")
		}

		var checkErr error
		if test == "ledger" {
			checkErr = runLedgerCheck(ctx, pg)
		}

		return errors.Join(k6Err, checkErr)
	},
}

func init() {
	loadtestCmd.Flags().String("port", "2100", "HTTP port of the load test server")
	loadtestCmd.Flags().Int("users", 0, "Override the bulk user count of the chosen test")
	loadtestCmd.Flags().Bool("serve", false, "Only start the server, and run k6 by hand")

	rootCmd.AddCommand(loadtestCmd)
}

func prepareLoadtestDatabase(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	db, err := database.ConnectToPostgres(ctx, cfg, logger.Named("database"))
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw SQL DB: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, sqlDB, "./storage/migrations"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if err := seeders.SeedDatabase(ctx, db, logger.Named("seeder"), cfg, "bulk"); err != nil {
		return fmt.Errorf("failed to seed database: %w", err)
	}
	return nil
}

func waitForLoadtestServer(ctx context.Context, baseURL string, appDone <-chan struct{}, appErr *error) error {
	client := &http.Client{Timeout: time.Second}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return errors.New("interrupted")
		case <-appDone:
			return errors.Join(fmt.Errorf("the app stopped before it was ready, see %s", loadtestLogFile), *appErr)
		case <-deadline:
			return fmt.Errorf("the app did not answer on %s/health within 30s, see %s", baseURL, loadtestLogFile)
		case <-ticker.C:
			res, err := client.Get(baseURL + "/health")
			if err != nil {
				continue
			}
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

func loadtestK6Env(test, baseURL string, users int) []string {
	env := []string{"BASE_URL=" + baseURL}
	if test == "ledger" {
		return append(env, "PG_CONTAINER="+loadtestContainer, "PG_DB="+loadtestDatabase, "PG_USER="+loadtestUser)
	}
	return append(env, fmt.Sprintf("SEED_USERS=%d", users))
}

func runK6(test string, env []string) error {
	// Not tied to the context: Ctrl+C reaches k6 from the terminal, and k6 stops
	// gracefully on its own. A kill from here would skip its summary.
	k6 := exec.Command("k6", "run", "scripts/k6/"+test+".js")
	k6.Env = append(os.Environ(), env...)
	k6.Stdin, k6.Stdout, k6.Stderr = os.Stdin, os.Stdout, os.Stderr

	err := k6.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 99 {
		return errors.New("k6 thresholds failed")
	}
	if err != nil {
		return fmt.Errorf("k6 failed: %w", err)
	}
	return nil
}

func runLedgerCheck(ctx context.Context, pg *postgres.PostgresContainer) error {
	const target = "/tmp/ledger_check.sql"
	if err := pg.CopyFileToContainer(ctx, loadtestCheckSQL, target, 0o644); err != nil {
		return fmt.Errorf("failed to copy the ledger check into the container: %w", err)
	}

	// The database is new, so every k6 ledger account in it belongs to this run
	run := "run=k6 ledger %"

	table, err := loadtestPsql(ctx, pg, "-v", run, "-f", target)
	if err != nil {
		return err
	}
	fmt.Printf("\nLedger check:\n%s", table)

	rows, err := loadtestPsql(ctx, pg, "-A", "-t", "-v", run, "-f", target)
	if err != nil {
		return err
	}
	rows = strings.TrimSpace(rows)
	if rows == "" {
		return errors.New("ledger check found no k6 ledger accounts")
	}
	for _, row := range strings.Split(rows, "\n") {
		cols := strings.Split(row, "|")
		if len(cols) < 3 {
			return fmt.Errorf("ledger check returned an unexpected row: %s", row)
		}
		for _, ok := range cols[len(cols)-3:] {
			if ok != "t" {
				return fmt.Errorf("ledger check failed: %s", row)
			}
		}
	}

	fmt.Println("Ledger check passed.")
	return nil
}

func loadtestPsql(ctx context.Context, pg *postgres.PostgresContainer, args ...string) (string, error) {
	cmd := append([]string{"psql", "-U", loadtestUser, "-d", loadtestDatabase}, args...)
	code, out, err := pg.Exec(ctx, cmd, tcexec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("failed to run psql: %w", err)
	}
	b, err := io.ReadAll(out)
	if err != nil {
		return "", fmt.Errorf("failed to read psql output: %w", err)
	}
	if code != 0 {
		return "", fmt.Errorf("psql exited with %d: %s", code, b)
	}
	return string(b), nil
}
