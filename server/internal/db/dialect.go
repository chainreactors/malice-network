package db

import (
	"gorm.io/gorm/clause"
)

// DialectAdapter encapsulates SQL dialect differences across database backends.
type DialectAdapter interface {
	// Name returns the dialect name.
	Name() string

	// FindAliveSessionsUpdateSQL returns the UPDATE SQL for marking inactive sessions.
	FindAliveSessionsUpdateSQL() string

	// FindAliveSessionsSelectSQL returns the SELECT SQL for querying alive sessions.
	FindAliveSessionsSelectSQL() string

	// AppendLogExpr returns the expression for appending a log entry (handles NULL + string concatenation).
	AppendLogExpr(logEntry string) clause.Expr

	// CastIDAsText returns a WHERE condition that casts the ID column to TEXT for LIKE prefix matching.
	CastIDAsText(column string) string

	// DateFunction returns the SQL expression for extracting the date part of a column.
	DateFunction(column string) string
}

// Adapter is the global dialect adapter, initialized in NewDBClient.
var Adapter DialectAdapter
