package db

import (
	"fmt"
	"strings"
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
		&models.Project{},
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
		cleanupPipelineNameIdentityArtifacts(dbClient, dbConfig.Dialect)
		addPostgresForeignKeys(dbClient)
	} else {
		dbClient.DisableForeignKeyConstraintWhenMigrating = true
		if err := dbClient.AutoMigrate(allModels...); err != nil {
			logs.Log.Warnf("Failed to auto-migrate database: %v", err)
		} else {
			logs.Log.Infof("database schema check completed (%s)", dbConfig.Dialect)
		}
		cleanupPipelineNameIdentityArtifacts(dbClient, dbConfig.Dialect)
		cleanupSQLiteLegacySchemaArtifacts(dbClient, allModels)
	}

	sqlDB, err := dbClient.DB()
	if err != nil {
		logs.Log.Errorf("Failed to get sql.DB: %v", err)
	} else {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	seedDefaultProject(dbClient)
	return dbClient, nil
}

func cleanupPipelineNameIdentityArtifacts(db *gorm.DB, dialect string) {
	var statements []string
	switch dialect {
	case configs.Postgres:
		statements = []string{
			`ALTER TABLE "contexts" DROP CONSTRAINT IF EXISTS "fk_contexts_pipeline"`,
			`ALTER TABLE "website_contents" DROP CONSTRAINT IF EXISTS "fk_website_contents_pipeline"`,
			`ALTER TABLE "pipelines" DROP CONSTRAINT IF EXISTS "uni_pipelines_name"`,
			`ALTER TABLE "pipelines" DROP CONSTRAINT IF EXISTS "pipelines_name_key"`,
			`DROP INDEX IF EXISTS "idx_pipelines_name"`,
			`DROP INDEX IF EXISTS "uni_pipelines_name"`,
		}
	default:
		statements = []string{
			`DROP INDEX IF EXISTS idx_pipelines_name`,
			`DROP INDEX IF EXISTS uni_pipelines_name`,
		}
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			logs.Log.Warnf("Failed to clean stale pipeline identity artifact: %v", err)
		}
	}
}

func cleanupSQLiteLegacySchemaArtifacts(db *gorm.DB, allModels []interface{}) {
	legacyTables := []struct {
		name    string
		model   interface{}
		markers []string
	}{
		{
			name:    "contexts",
			model:   &models.Context{},
			markers: []string{"fk_contexts_pipeline"},
		},
		{
			name:    "website_contents",
			model:   &models.WebsiteContent{},
			markers: []string{"fk_website_contents_pipeline"},
		},
		{
			name:    "sessions",
			model:   &models.Session{},
			markers: []string{"fk_tasks_session", "fk_contexts_session"},
		},
		{
			name:    "pipelines",
			model:   &models.Pipeline{},
			markers: []string{"uni_pipelines_name", "UNIQUE (`name`)", "UNIQUE (name)", "UNIQUE (\"name\")"},
		},
	}

	rebuilt := false
	for _, table := range legacyTables {
		ddl, err := sqliteTableDDL(db, table.name)
		if err != nil {
			logs.Log.Warnf("Failed to inspect SQLite table %s: %v", table.name, err)
			continue
		}
		if !containsAny(ddl, table.markers) {
			continue
		}
		if err := rebuildSQLiteTable(db, table.name, table.model); err != nil {
			logs.Log.Warnf("Failed to rebuild legacy SQLite table %s: %v", table.name, err)
			continue
		}
		rebuilt = true
	}

	if rebuilt {
		if err := db.AutoMigrate(allModels...); err != nil {
			logs.Log.Warnf("Failed to finalize SQLite schema cleanup: %v", err)
		}
	}
}

func sqliteTableDDL(db *gorm.DB, table string) (string, error) {
	var ddl string
	err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", table).Row().Scan(&ddl)
	return ddl, err
}

func containsAny(s string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

func rebuildSQLiteTable(db *gorm.DB, table string, model interface{}) error {
	tempTable := fmt.Sprintf("__legacy_%s_%d", table, time.Now().UnixNano())
	if err := db.Exec("PRAGMA foreign_keys=OFF").Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		oldColumns, err := sqliteColumnNames(tx, table)
		if err != nil {
			return err
		}
		if err := tx.Exec(
			fmt.Sprintf("ALTER TABLE %s RENAME TO %s", quoteSQLiteIdentifier(table), quoteSQLiteIdentifier(tempTable)),
		).Error; err != nil {
			return err
		}
		if err := dropSQLiteUserIndexes(tx, tempTable); err != nil {
			return err
		}
		if err := tx.Migrator().CreateTable(model); err != nil {
			return err
		}
		newColumns, err := sqliteColumnNames(tx, table)
		if err != nil {
			return err
		}
		commonColumns := commonSQLiteColumns(oldColumns, newColumns)
		if len(commonColumns) > 0 {
			quotedColumns := quoteSQLiteIdentifiers(commonColumns)
			if err := tx.Exec(fmt.Sprintf(
				"INSERT INTO %s (%s) SELECT %s FROM %s",
				quoteSQLiteIdentifier(table),
				quotedColumns,
				quotedColumns,
				quoteSQLiteIdentifier(tempTable),
			)).Error; err != nil {
				return err
			}
		}
		return tx.Exec(fmt.Sprintf("DROP TABLE %s", quoteSQLiteIdentifier(tempTable))).Error
	})
}

func sqliteColumnNames(db *gorm.DB, table string) ([]string, error) {
	rows, err := db.Raw("SELECT name FROM pragma_table_info(?)", table).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}

func dropSQLiteUserIndexes(db *gorm.DB, table string) error {
	rows, err := db.Raw("SELECT name FROM pragma_index_list(?) WHERE origin='c'", table).Rows()
	if err != nil {
		return err
	}

	var indexes []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		indexes = append(indexes, name)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, name := range indexes {
		if err := db.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", quoteSQLiteIdentifier(name))).Error; err != nil {
			return err
		}
	}
	return nil
}

func commonSQLiteColumns(oldColumns, newColumns []string) []string {
	oldSet := make(map[string]struct{}, len(oldColumns))
	for _, column := range oldColumns {
		oldSet[column] = struct{}{}
	}
	common := make([]string, 0, len(newColumns))
	for _, column := range newColumns {
		if _, ok := oldSet[column]; ok {
			common = append(common, column)
		}
	}
	return common
}

func quoteSQLiteIdentifiers(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, quoteSQLiteIdentifier(name))
	}
	return strings.Join(quoted, ", ")
}

func quoteSQLiteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
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
		name   string
		table  string
		column string
		refTab string
		refCol string
	}{
		{"fk_sessions_profile", "sessions", "profile_name", "profiles", "name"},
		{"fk_tasks_session", "tasks", "session_id", "sessions", "session_id"},
		{"fk_contexts_session", "contexts", "session_id", "sessions", "session_id"},
		{"fk_contexts_task", "contexts", "task_id", "tasks", "id"},
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

func seedDefaultProject(dbClient *gorm.DB) {
	var count int64
	dbClient.Model(&models.Project{}).Where("name = ?", "default").Count(&count)
	if count == 0 {
		project := &models.Project{
			Name:        "default",
			Description: "Default project",
		}
		if err := dbClient.Create(project).Error; err != nil {
			logs.Log.Warnf("Failed to create default project: %v", err)
		} else {
			logs.Log.Infof("default project created")
		}
	}
}
