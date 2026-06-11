package configs

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
)

const (
	// Sqlite - SQLite protocol
	Sqlite = "sqlite3"
	// Postgres - Postgresql protocol
	Postgres = "postgresql"
	// MySQL - MySQL protocol
	MySQL = "mysql"
)

var (
	// ErrInvalidDialect - An invalid dialect was specified
	ErrInvalidDialect = errors.New("invalid SQL Dialect")
	databaseFileName  = filepath.Join(ServerRootPath, "malice.db")
)

// DatabaseConfig - Database configuration
type DatabaseConfig struct {
	Dialect string `json:"dialect" config:"dialect" default:"sqlite3" yaml:"dialect"`
	Database string `json:"database" config:"database" yaml:"database"`
	Username string `json:"username" config:"username" yaml:"username"`
	Password string `json:"password" config:"password" yaml:"password"`
	Host     string `json:"host" config:"host" yaml:"host"`
	Port     uint16 `json:"port" config:"port" yaml:"port"`

	Params map[string]string `json:"params" config:"params" yaml:"params"`

	MaxIdleConns int `json:"max_idle_conns" config:"max_idle_conns" default:"10" yaml:"max_idle_conns"`
	MaxOpenConns int `json:"max_open_conns" config:"max_open_conns" default:"100" yaml:"max_open_conns"`

	LogLevel string `json:"log_level" config:"log_level" default:"warn" yaml:"log_level"`
}

// DSN - Get the db connections string
func (c *DatabaseConfig) DSN() (string, error) {
	switch c.Dialect {
	case Sqlite:
		filePath := databaseFileName
		params := encodeParams(c.Params)
		return fmt.Sprintf("file:%s?%s", filePath, params), nil
	case Postgres:
		host := c.Host
		if host == "" {
			host = "localhost"
		}
		port := c.Port
		if port == 0 {
			port = 5432
		}
		db := c.Database
		if db == "" {
			db = "malice"
		}
		params := encodeParamsPostgres(c.Params)
		logs.Log.Infof("Connecting to PostgreSQL database %s@%s:%d/%s", c.Username, host, port, db)
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s",
			host, port, c.Username, c.Password, db, params), nil
	default:
		return "", ErrInvalidDialect
	}
}

func encodeParams(rawParams map[string]string) string {
	params := url.Values{}
	for key, value := range rawParams {
		params.Add(key, value)
	}
	return params.Encode()
}

// encodeParamsPostgres encodes params in space-separated key=value format for PostgreSQL DSN
func encodeParamsPostgres(rawParams map[string]string) string {
	var parts []string
	for key, value := range rawParams {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}

// GetDefaultDatabaseConfig returns the default database configuration (SQLite)
func GetDefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Dialect:      Sqlite,
		MaxIdleConns: 10,
		MaxOpenConns: 100,
		LogLevel:     "warn",
	}
}
