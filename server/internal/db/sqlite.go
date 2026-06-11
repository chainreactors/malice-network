package db

import (
	"context"
	"database/sql"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm/callbacks"

	_ "github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// DriverName is the default driver name for SQLite.
const DriverName = "sqlite3"

type Migrator struct {
	migrator.Migrator
}

type Dialector struct {
	DriverName string
	DSN        string
	Conn       gorm.ConnPool
}

func Open(dsn string) gorm.Dialector {
	return &Dialector{DSN: dsn}
}

func (dialector Dialector) Name() string {
	return "sqlite"
}

func (dialector Dialector) Initialize(db *gorm.DB) (err error) {
	if dialector.DriverName == "" {
		dialector.DriverName = DriverName
	}

	if dialector.Conn != nil {
		db.ConnPool = dialector.Conn
	} else {
		conn, err := sql.Open(dialector.DriverName, dialector.DSN)
		if err != nil {
			return err
		}
		db.ConnPool = conn
	}

	var version string
	if err := db.ConnPool.QueryRowContext(context.Background(), "select sqlite_version()").Scan(&version); err != nil {
		return err
	}
	// https://www.sqlite.org/releaselog/3_35_0.html
	if compareVersion(version, "3.35.0") >= 0 {
		callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
			CreateClauses:        []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
			UpdateClauses:        []string{"UPDATE", "SET", "WHERE", "RETURNING"},
			DeleteClauses:        []string{"DELETE", "FROM", "WHERE", "RETURNING"},
			LastInsertIDReversed: true,
		})
	} else {
		callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
			LastInsertIDReversed: true,
		})
	}

	for k, v := range dialector.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}
	return
}

func (dialector Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	return map[string]clause.ClauseBuilder{
		"INSERT": func(c clause.Clause, builder clause.Builder) {
			if insert, ok := c.Expression.(clause.Insert); ok {
				if stmt, ok := builder.(*gorm.Statement); ok {
					stmt.WriteString("INSERT ")
					if insert.Modifier != "" {
						stmt.WriteString(insert.Modifier)
						stmt.WriteByte(' ')
					}

					stmt.WriteString("INTO ")
					if insert.Table.Name == "" {
						stmt.WriteQuoted(stmt.Table)
					} else {
						stmt.WriteQuoted(insert.Table)
					}
					return
				}
			}

			c.Build(builder)
		},
		"LIMIT": func(c clause.Clause, builder clause.Builder) {
			if limit, ok := c.Expression.(clause.Limit); ok {
				var lmt = -1
				if limit.Limit != nil && *limit.Limit >= 0 {
					lmt = *limit.Limit
				}
				if lmt >= 0 || limit.Offset > 0 {
					builder.WriteString("LIMIT ")
					builder.WriteString(strconv.Itoa(lmt))
				}
				if limit.Offset > 0 {
					builder.WriteString(" OFFSET ")
					builder.WriteString(strconv.Itoa(limit.Offset))
				}
			}
		},
		"FOR": func(c clause.Clause, builder clause.Builder) {
			if _, ok := c.Expression.(clause.Locking); ok {
				// SQLite3 does not support row-level locking.
				return
			}
			c.Build(builder)
		},
	}
}

func (dialector Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	if field.AutoIncrement {
		return clause.Expr{SQL: "NULL"}
	}

	// doesn't work, will raise error
	return clause.Expr{SQL: "DEFAULT"}
}

func (dialector Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   dialector,
		CreateIndexAfterCreateTable: true,
	}}}
}

// HasTable checks table existence via sqlite_master instead of information_schema.
func (m Migrator) HasTable(value interface{}) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
			stmt.Table,
		).Row().Scan(&count)
	})
	return count > 0
}

// HasColumn checks column existence via PRAGMA table_info.
func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		name := field
		if stmt.Schema != nil {
			if f := stmt.Schema.LookUpField(field); f != nil {
				name = f.DBName
			}
		}
		return m.DB.Raw(
			"SELECT count(*) FROM pragma_table_info(?) WHERE name=?",
			stmt.Table, name,
		).Row().Scan(&count)
	})
	return count > 0
}

// HasIndex checks index existence via PRAGMA index_list.
func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}
		return m.DB.Raw(
			"SELECT count(*) FROM pragma_index_list(?) WHERE name=?",
			stmt.Table, name,
		).Row().Scan(&count)
	})
	return count > 0
}

// CurrentDatabase returns empty string for SQLite (single-database system).
func (m Migrator) CurrentDatabase() (name string) {
	return ""
}

// AlterColumn is a no-op for SQLite. SQLite does not support ALTER COLUMN;
// the official GORM sqlite driver uses full table recreation, but for our
// use-case schema-level column type changes are not expected at runtime.
func (m Migrator) AlterColumn(value interface{}, field string) error {
	return nil
}

// CreateConstraint is a no-op for SQLite. SQLite only supports foreign key
// constraints defined at table creation time, not added via ALTER TABLE.
func (m Migrator) CreateConstraint(value interface{}, name string) error {
	return nil
}

// HasConstraint checks constraint existence via sqlite_master DDL parsing.
func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=? AND sql LIKE ?",
			stmt.Table, "%"+name+"%",
		).Row().Scan(&count)
	})
	return count > 0
}

// ColumnTypes returns column type information via pragma_table_info (table-valued function),
// which supports parameterized queries and avoids SQL injection.
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var columns []struct {
			Name    string  `gorm:"column:name"`
			Type    string  `gorm:"column:type"`
			NotNull int64   `gorm:"column:notnull"`
			Dflt    *string `gorm:"column:dflt_value"`
			Pk      int64   `gorm:"column:pk"`
		}
		if err := m.DB.Raw("SELECT * FROM pragma_table_info(?)", stmt.Table).Scan(&columns).Error; err != nil {
			// fallback to default implementation
			rows, err2 := m.DB.Session(&gorm.Session{}).Table(stmt.Table).Limit(1).Rows()
			if err2 != nil {
				return err2
			}
			defer rows.Close()
			rawCols, err2 := rows.ColumnTypes()
			if err2 != nil {
				return err2
			}
			for _, c := range rawCols {
				columnTypes = append(columnTypes, migrator.ColumnType{SQLColumnType: c})
			}
			return nil
		}
		for _, col := range columns {
			nullable := col.NotNull == 0
			isPrimary := col.Pk > 0
			ct := sqliteColumnType{
				name:         col.Name,
				dataType:     col.Type,
				nullable:     &nullable,
				primaryKey:   &isPrimary,
				defaultValue: col.Dflt,
			}
			columnTypes = append(columnTypes, ct)
		}
		return nil
	})
	return columnTypes, err
}

func (dialector Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (dialector Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				writer.WriteString(".`")
			}
			writer.WriteString(str)
			writer.WriteByte('`')
		}
	} else {
		writer.WriteString(str)
		writer.WriteByte('`')
	}
}

func (dialector Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (dialector Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "numeric"
	case schema.Int, schema.Uint:
		if field.AutoIncrement && !field.PrimaryKey {
			// https://www.sqlite.org/autoinc.html
			return "integer PRIMARY KEY AUTOINCREMENT"
		} else {
			return "integer"
		}
	case schema.Float:
		return "real"
	case schema.String:
		return "text"
	case schema.Time:
		return "datetime"
	case schema.Bytes:
		return "blob"
	}

	return string(field.DataType)
}

func (dialectopr Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (dialectopr Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}

func compareVersion(version1, version2 string) int {
	n, m := len(version1), len(version2)
	i, j := 0, 0
	for i < n || j < m {
		x := 0
		for ; i < n && version1[i] != '.'; i++ {
			x = x*10 + int(version1[i]-'0')
		}
		i++
		y := 0
		for ; j < m && version2[j] != '.'; j++ {
			y = y*10 + int(version2[j]-'0')
		}
		j++
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

// sqliteColumnType implements gorm.ColumnType for SQLite PRAGMA results.
type sqliteColumnType struct {
	name         string
	dataType     string
	nullable     *bool
	primaryKey   *bool
	defaultValue *string
}

func (c sqliteColumnType) Name() string                                   { return c.name }
func (c sqliteColumnType) DatabaseTypeName() string                       { return c.dataType }
func (c sqliteColumnType) ColumnType() (string, bool)                     { return c.dataType, true }
func (c sqliteColumnType) PrimaryKey() (bool, bool)                       { if c.primaryKey != nil { return *c.primaryKey, true }; return false, false }
func (c sqliteColumnType) AutoIncrement() (bool, bool)                    { return false, false }
func (c sqliteColumnType) Length() (int64, bool)                          { return 0, false }
func (c sqliteColumnType) DecimalSize() (int64, int64, bool)              { return 0, 0, false }
func (c sqliteColumnType) Nullable() (bool, bool)                         { if c.nullable != nil { return *c.nullable, true }; return true, false }
func (c sqliteColumnType) Unique() (bool, bool)                           { return false, false }
func (c sqliteColumnType) ScanType() reflect.Type                         { return nil }
func (c sqliteColumnType) Comment() (string, bool)                        { return "", false }
func (c sqliteColumnType) DefaultValue() (string, bool)                   { if c.defaultValue != nil { return *c.defaultValue, true }; return "", false }
