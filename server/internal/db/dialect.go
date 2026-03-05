package db

import (
	"gorm.io/gorm/clause"
)

// DialectAdapter 封装不同数据库方言的 SQL 差异
type DialectAdapter interface {
	// Name 返回方言名称
	Name() string

	// FindAliveSessionsUpdateSQL 返回标记不活跃会话的 UPDATE SQL
	FindAliveSessionsUpdateSQL() string

	// FindAliveSessionsSelectSQL 返回查询活跃会话的 SELECT SQL
	FindAliveSessionsSelectSQL() string

	// AppendLogExpr 返回追加日志的表达式（处理 NULL + 字符串拼接）
	AppendLogExpr(logEntry string) clause.Expr

	// CastIDAsText 返回将 ID 列转换为 TEXT 做 LIKE 前缀匹配的 WHERE 条件
	CastIDAsText(column string) string

	// DateFunction 返回提取日期部分的 SQL 表达式
	DateFunction(column string) string
}

// Adapter 全局方言适配器，在 NewDBClient 中初始化
var Adapter DialectAdapter
