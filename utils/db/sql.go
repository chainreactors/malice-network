package db

import (
	"fmt"
	"github.com/chainreactors/malice-network/utils/configs"
	"time"

	"github.com/chainreactors/malice-network/utils/db/models"
	"gorm.io/driver/sqlite"
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

	err := dbClient.AutoMigrate(
		&models.Beacon{},
		&models.BeaconTask{},
		&models.DNSCanary{},
		&models.Crackstation{},
		&models.Benchmark{},
		&models.CrackTask{},
		&models.CrackCommand{},
		&models.CrackFile{},
		&models.CrackFileChunk{},
		&models.Certificate{},
		&models.Host{},
		&models.IOC{},
		&models.ExtensionData{},
		&models.ImplantBuild{},
		&models.ImplantProfile{},
		&models.ImplantConfig{},
		&models.ImplantC2{},
		&models.EncoderAsset{},
		&models.KeyExHistory{},
		&models.KeyValue{},
		&models.CanaryDomain{},
		&models.Loot{},
		&models.Credential{},
		&models.Operator{},
		&models.Website{},
		&models.WebContent{},
		&models.WGKeys{},
		&models.WGPeer{},
	)
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

func sqliteClient(dbConfiug *configs.DatabaseConfig) *gorm.DB {
	// 连接 SQLite 数据库
	dbClient, err := gorm.Open(sqlite.Open("sliver.db"), &gorm.Config{
		PrepareStmt: true,
		//Logger:      getGormLogger(dbConfig),
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
