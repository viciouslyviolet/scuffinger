package services

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	selftest "scuffinger/database/self_test"
	"scuffinger/internal/config"
	"scuffinger/internal/i18n"
	"scuffinger/internal/logging"
)

// DatabaseService manages the PostgreSQL database connection.
type DatabaseService struct {
	cfg  config.DatabaseConfig
	pool *pgxpool.Pool
	log  *logging.Logger
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(cfg config.DatabaseConfig, log *logging.Logger) *DatabaseService {
	return &DatabaseService{cfg: cfg, log: log}
}

func (s *DatabaseService) Name() string { return "database" }

func (s *DatabaseService) Connect(ctx context.Context) error {
	s.log.Debug("Dialing database", "host", s.cfg.Host, "port", s.cfg.Port, "db", s.cfg.Name)
	pool, err := pgxpool.New(ctx, s.connString(s.cfg.Name))
	if err != nil {
		return i18n.Err(i18n.ErrDbConnect, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return i18n.Err(i18n.ErrDbPing, err)
	}
	s.pool = pool
	return nil
}

// SelfTest creates a temporary "testdb" database, runs full CRUD operations
// inside it, then drops it.
func (s *DatabaseService) SelfTest(ctx context.Context) error {
	const testDB = "testdb"

	// Cleanup any leftover testdb from a previous crashed run.
	// We need to terminate existing connections first.
	_, _ = s.pool.Exec(ctx, fmt.Sprintf(selftest.TerminateConnections, testDB))
	_, _ = s.pool.Exec(ctx, fmt.Sprintf(selftest.DropDatabase, testDB))

	// 1. Create the test database
	s.log.Debug(i18n.Get(i18n.MsgDbCreatingTestDb), "database", testDB)
	if _, err := s.pool.Exec(ctx, fmt.Sprintf(selftest.CreateDatabase, testDB)); err != nil {
		return i18n.Err(i18n.ErrDbCreateTestDb, err)
	}

	// 2. Connect to testdb
	testPool, err := pgxpool.New(ctx, s.connString(testDB))
	if err != nil {
		_ = s.dropDB(ctx, testDB)
		return i18n.Err(i18n.ErrDbConnectTestDb, err)
	}

	// 3. Run CRUD
	s.log.Debug(i18n.Get(i18n.MsgDbRunningCrud))
	crudErr := s.runCRUD(ctx, testPool)

	// 4. Close the testdb connection before dropping
	testPool.Close()

	// 5. Drop testdb
	if dropErr := s.dropDB(ctx, testDB); dropErr != nil {
		s.log.Warn(i18n.Get(i18n.WarnDbDropTestFailed), "database", testDB, "error", dropErr)
	}

	if crudErr != nil {
		return i18n.Err(i18n.ErrDbCrud, crudErr)
	}

	s.log.Info(i18n.Get(i18n.MsgDbSelfTestPassed))
	return nil
}

func (s *DatabaseService) runCRUD(ctx context.Context, pool *pgxpool.Pool) error {
	// CREATE TABLE
	if _, err := pool.Exec(ctx, selftest.CreateSelfTestTable); err != nil {
		return i18n.Err(i18n.ErrDbCreateTable, err)
	}

	// INSERT
	if _, err := pool.Exec(ctx, selftest.InsertSelfTest, "test_key", "test_value"); err != nil {
		return i18n.Err(i18n.ErrDbInsert, err)
	}

	// READ
	var value string
	if err := pool.QueryRow(ctx, selftest.SelectSelfTest, "test_key").Scan(&value); err != nil {
		return i18n.Err(i18n.ErrDbRead, err)
	}
	if value != "test_value" {
		return fmt.Errorf("%s: got %q, want %q", i18n.Get(i18n.ErrDbReadMismatch), value, "test_value")
	}

	// UPDATE
	if _, err := pool.Exec(ctx, selftest.UpdateSelfTest, "updated_value", "test_key"); err != nil {
		return i18n.Err(i18n.ErrDbUpdate, err)
	}

	// READ AGAIN
	if err := pool.QueryRow(ctx, selftest.SelectSelfTest, "test_key").Scan(&value); err != nil {
		return i18n.Err(i18n.ErrDbReadAfterUpdate, err)
	}
	if value != "updated_value" {
		return fmt.Errorf("%s: got %q, want %q", i18n.Get(i18n.ErrDbUpdateMismatch), value, "updated_value")
	}

	return nil
}

func (s *DatabaseService) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *DatabaseService) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// Pool returns the underlying connection pool for reuse by other packages.
func (s *DatabaseService) Pool() *pgxpool.Pool {
	return s.pool
}

// connString builds a PostgreSQL connection string for the given database name.
func (s *DatabaseService) connString(dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		s.cfg.Host, s.cfg.Port, s.cfg.User, s.cfg.Password, dbName, s.cfg.SSLMode,
	)
}

// dropDB terminates connections to the target database and drops it.
func (s *DatabaseService) dropDB(ctx context.Context, name string) error {
	_, _ = s.pool.Exec(ctx, fmt.Sprintf(selftest.TerminateConnections, name))
	_, err := s.pool.Exec(ctx, fmt.Sprintf(selftest.DropDatabase, name))
	return err
}
