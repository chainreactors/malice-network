package db

import (
	"gorm.io/gorm/clause"
)

type postgresAdapter struct{}

func (a *postgresAdapter) Name() string { return "postgres" }

func (a *postgresAdapter) FindAliveSessionsUpdateSQL() string {
	return `UPDATE sessions
		SET is_alive = false
		WHERE last_checkin < EXTRACT(EPOCH FROM NOW())::bigint - (
			COALESCE(
				CASE WHEN data IS NOT NULL AND data != ''
					THEN (data::json->>'interval')::integer
				END,
				30
			) * 2
		)
		AND is_removed = false`
}

func (a *postgresAdapter) FindAliveSessionsSelectSQL() string {
	return `SELECT *
		FROM sessions
		WHERE last_checkin > EXTRACT(EPOCH FROM NOW())::bigint - (
			COALESCE(
				CASE WHEN data IS NOT NULL AND data != ''
					THEN (data::json->>'interval')::integer
				END,
				30
			) * 2
		)
		AND is_removed = false`
}

func (a *postgresAdapter) AppendLogExpr(logEntry string) clause.Expr {
	return clause.Expr{SQL: "COALESCE(log, '') || ?", Vars: []interface{}{logEntry}}
}

func (a *postgresAdapter) CastIDAsText(column string) string {
	return column + "::text LIKE ?"
}

func (a *postgresAdapter) DateFunction(column string) string {
	return column + "::date"
}
