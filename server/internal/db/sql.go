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

// NewDBClient - Initialize the db client
func NewDBClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
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
	switch dbConfig.Dialect {
	case configs.Sqlite:
		Adapter = &sqliteAdapter{}
		dbClient = sqliteClient(dbConfig)
	case configs.Postgres:
		Adapter = &postgresAdapter{}
		dbClient = postgresClient(dbConfig)
	default:
		panic(fmt.Sprintf("Unknown DB Dialect: '%s'", dbConfig.Dialect))
	}
	if err := dbClient.AutoMigrate(
		&models.Profile{},
		&models.Artifact{},
		&models.Session{},
		&models.Pipeline{},
		&models.Task{},
		&models.WebsiteContent{},
		&models.Operator{},
		&models.Certificate{},
		&models.Context{},
		&models.AuthzRule{},
	); err != nil {
		logs.Log.Warnf("Failed to auto-migrate database: %v", err)
	}

	sqlDB, err := dbClient.DB()
	if err != nil {
		logs.Log.Errorf("Failed to get sql.DB: %v", err)
	} else {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	return dbClient
}

func sqliteClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate SQLite DSN: %v", err))
	}
	dbClient, err := gorm.Open(Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to open SQLite database: %v", err))
	}
	return dbClient
}

func postgresClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate PostgreSQL DSN: %v", err))
	}
	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to open PostgreSQL database: %v", err))
	}
	return dbClient
}
