package db

import (
	"gorm.io/gorm"
)

// Client - Database Client
var (
	// ErrRecordNotFound - Record not found error
	ErrRecordNotFound = gorm.ErrRecordNotFound
	Client            = newDBClient()
)

// Session - Database session
func Session() *gorm.DB {
	return Client.Session(&gorm.Session{
		FullSaveAssociations: true,
		PrepareStmt:          true,
	})
}
