package db

import (
	"fmt"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm/logger"

	"gorm.io/gorm"
)

// NewDBClient initializes the db client. Returns an error instead of panicking
// on configuration or connection failures.
func NewDBClient(dbConfig *configs.DatabaseConfig) (*gorm.DB, error) {
	if dbConfig == nil {
		dbConfig = configs.GetDefaultDatabaseConfig()
	}
	if dbConfig.Dialect == "" {
		dbConfig.Dialect = configs.Sqlite
	}
	if dbConfig.MaxIdleConns < 1 {
		dbConfig.MaxIdleConns = 1
	}
	if dbConfig.MaxOpenConns < 1 {
		dbConfig.MaxOpenConns = 1
	}
	var dbClient *gorm.DB
	var err error
	switch dbConfig.Dialect {
	case configs.Sqlite:
		Adapter = &sqliteAdapter{}
		dbClient, err = sqliteClient(dbConfig)
	case configs.Postgres:
		Adapter = &postgresAdapter{}
		dbClient, err = postgresClient(dbConfig)
	default:
		return nil, fmt.Errorf("unknown DB dialect: %q", dbConfig.Dialect)
	}
	if err != nil {
		return nil, err
	}

	allModels := []interface{}{
		&models.Pipeline{},
		&models.Operator{},
		&models.Certificate{},
		&models.AuthzRule{},
		&models.Profile{},
		&models.WebsiteContent{},
		&models.Session{},
		&models.Artifact{},
		&models.Task{},
		&models.Context{},
	}

	if dbConfig.Dialect == configs.Postgres {
		// PostgreSQL: two-pass migration.
		// Pass 1: create all tables without FK constraints.
		// Pass 2: add FK constraints via raw SQL to avoid GORM's auto-detected
		// reverse relationships generating incorrect FK direction (e.g., when
		// Session and Task both have a SessionID field, GORM may generate
		// sessions.session_id → tasks.session_id instead of the correct reverse).
		dbClient.DisableForeignKeyConstraintWhenMigrating = true
		if err := dbClient.AutoMigrate(allModels...); err != nil {
			logs.Log.Warnf("Failed to create tables: %v", err)
		} else {
			logs.Log.Infof("database schema check completed (%s)", dbConfig.Dialect)
		}
		addPostgresForeignKeys(dbClient)
	} else {
		if err := dbClient.AutoMigrate(allModels...); err != nil {
			logs.Log.Warnf("Failed to auto-migrate database: %v", err)
		} else {
			logs.Log.Infof("database schema check completed (%s)", dbConfig.Dialect)
		}
	}

	sqlDB, err := dbClient.DB()
	if err != nil {
		logs.Log.Errorf("Failed to get sql.DB: %v", err)
	} else {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	return dbClient, nil
}

func sqliteClient(dbConfig *configs.DatabaseConfig) (*gorm.DB, error) {
	dsn, err := dbConfig.DSN()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQLite DSN: %w", err)
	}
	dbClient, err := gorm.Open(Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	return dbClient, nil
}

func postgresClient(dbConfig *configs.DatabaseConfig) (*gorm.DB, error) {
	dsn, err := dbConfig.DSN()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PostgreSQL DSN: %w", err)
	}
	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}
	return dbClient, nil
}

// addPostgresForeignKeys adds FK constraints via raw SQL.
// This avoids GORM's auto-detected reverse relationships which can generate
// incorrect FK direction when both sides share the same field name (e.g. SessionID).
func addPostgresForeignKeys(db *gorm.DB) {
	fks := []struct {
		name    string
		table   string
		column  string
		refTab  string
		refCol  string
	}{
		{"fk_sessions_profile", "sessions", "profile_name", "profiles", "name"},
		{"fk_tasks_session", "tasks", "session_id", "sessions", "session_id"},
		{"fk_contexts_session", "contexts", "session_id", "sessions", "session_id"},
		{"fk_contexts_pipeline", "contexts", "pipeline_id", "pipelines", "name"},
		{"fk_contexts_task", "contexts", "task_id", "tasks", "id"},
		{"fk_website_contents_pipeline", "website_contents", "pipeline_id", "pipelines", "name"},
	}

	for _, fk := range fks {
		// Skip if constraint already exists (idempotent for existing databases)
		var count int64
		db.Raw(
			"SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = CURRENT_SCHEMA() AND table_name = ? AND constraint_name = ?",
			fk.table, fk.name,
		).Scan(&count)
		if count > 0 {
			continue
		}

		sql := fmt.Sprintf(
			`ALTER TABLE "%s" ADD CONSTRAINT "%s" FOREIGN KEY ("%s") REFERENCES "%s"("%s") ON UPDATE CASCADE ON DELETE SET NULL`,
			fk.table, fk.name, fk.column, fk.refTab, fk.refCol,
		)
		if err := db.Exec(sql).Error; err != nil {
			logs.Log.Warnf("Failed to add FK %s: %v", fk.name, err)
		}
	}
}
