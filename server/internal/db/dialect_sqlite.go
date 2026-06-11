package db

import (
	"gorm.io/gorm/clause"
)

type sqliteAdapter struct{}

func (a *sqliteAdapter) Name() string { return "sqlite" }

func (a *sqliteAdapter) FindAliveSessionsUpdateSQL() string {
	return `UPDATE sessions
		SET is_alive = false
		WHERE last_checkin < strftime('%s', 'now') - (
			CAST(COALESCE(
				JSON_EXTRACT(data, '$.interval'),
				'30'
			) AS INTEGER) * 2
		)
		AND is_removed = false`
}

func (a *sqliteAdapter) FindAliveSessionsSelectSQL() string {
	return `SELECT *
		FROM sessions
		WHERE last_checkin > strftime('%s', 'now') - (
			CAST(COALESCE(
				JSON_EXTRACT(data, '$.interval'),
				'30'
			) AS INTEGER) * 2
		)
		AND is_removed = false`
}

func (a *sqliteAdapter) AppendLogExpr(logEntry string) clause.Expr {
	return clause.Expr{SQL: "ifnull(log, '') || ?", Vars: []interface{}{logEntry}}
}

func (a *sqliteAdapter) CastIDAsText(column string) string {
	return "CAST(" + column + " AS TEXT) LIKE ?"
}

func (a *sqliteAdapter) DateFunction(column string) string {
	return "DATE(" + column + ")"
}
