package db

import (
	"fmt"
	"github.com/chainreactors/malice-network/server/configs"
	"time"

	"gorm.io/gorm"
)

// newDBClient - Initialize the db client
func newDBClient() *gorm.DB {
	dbConfig := configs.GetDatabaseConfig()

	var dbClient *gorm.DB
	switch dbConfig.Dialect {
	case configs.Sqlite:
		dbClient = sqliteClient(dbConfig)
	default:
		panic(fmt.Sprintf("Unknown DB Dialect: '%s'", dbConfig.Dialect))
	}

	err := dbClient.AutoMigrate()
	if err != nil {
		// TODO -log client error
		//clientLog.Error(err)
	}

	// Get generic database object sql.DB to use its functions
	sqlDB, err := dbClient.DB()
	if err != nil {
		// TODO - log client error
		//clientLog.Error(err)
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	return dbClient
}

func sqliteClient(dbConfig *configs.DatabaseConfig) *gorm.DB {
	dsn, err := dbConfig.DSN()
	if err != nil {
		panic(err)
	}
	dbClient, err := gorm.Open(Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic(err)
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
