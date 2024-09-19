package db

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"gorm.io/gorm/logger"
	"time"

	"gorm.io/gorm"
)

// newDBClient - Initialize the db client
func NewDBClient() *gorm.DB {
	dbConfig := configs.GetDatabaseConfig()
	var dbClient *gorm.DB
	switch dbConfig.Dialect {
	case configs.Sqlite:
		dbClient = sqliteClient(dbConfig)
	default:
		panic(fmt.Sprintf("Unknown DB Dialect: '%s'", dbConfig.Dialect))
	}
	_ = dbClient.AutoMigrate(
		&models.WebsiteContent{},
		&models.Pipeline{},
		&models.Operator{},
		&models.Certificate{},
		&models.Session{},
		&models.Task{},
	)
	if dbClient == nil {
		logs.Log.Errorf("Failed to initialize database")
	} else {
		// Get generic database object sql.DB to use its functions
		sqlDB, err := dbClient.DB()
		if err != nil {
			logs.Log.Errorf("Failed to get sql.DB: %v", err)
		}
		// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

		// SetMaxOpenConns sets the maximum number of open connections to the database.
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

		// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	return dbClient
}

func sqliteClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		logs.Log.Errorf("Failed to get DSN: %v", err)
	}
	dbClient, err := gorm.Open(Open(dsn), &gorm.Config{
		PrepareStmt: false,
		Logger:      logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		logs.Log.Errorf("Failed to open sqlite: %v", err)
	}
	return dbClient
}

//func postgresClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
//	dsn, err := dbConfig.DSN()
//	if err != nil {
//		panic(err)
//	}
//	dbClient, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
//		PrepareStmt: true,
//		Logger:      getGormLogger(dbConfig),
//	})
//	if err != nil {
//		panic(err)
//	}
//	return dbClient
//}
//
//func mySQLClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
//	dsn, err := dbConfig.DSN()
//	if err != nil {
//		panic(err)
//	}
//	dbClient, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
//		PrepareStmt: true,
//		Logger:      getGormLogger(dbConfig),
//	})
//	if err != nil {
//		panic(err)
//	}
//	return dbClient
//}
