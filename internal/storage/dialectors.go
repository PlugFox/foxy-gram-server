package storage

import (
	"fmt"
	"strings"

	config "github.com/plugfox/foxy-gram-server/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var errorUnsupportedDriver = fmt.Errorf("unsupported database driver")

// createDialector creates the appropriate GORM dialector based on the config.
func createDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	switch strings.ToLower(cfg.Driver) {
	case "sqlite3":
		return sqliteDialector(cfg.Connection)
	case "sqlite":
		return sqliteDialector(cfg.Connection)
	case "postgres":
		return postgresDialector(cfg.Connection)
	case "mysql":
		return mysqlDialector(cfg.Connection)
	case "mariadb":
		return mysqlDialector(cfg.Connection)
	case "tidb":
		return mysqlDialector(cfg.Connection)
	default:
		return nil, errorUnsupportedDriver
	}
}

func sqliteDialector(connection string) (gorm.Dialector, error) {
	if connection == ":memory:" {
		return sqlite.Open("file::memory:?cache=shared"), nil
	}
	return sqlite.Open(connection), nil
}

func postgresDialector(connection string) (gorm.Dialector, error) {
	return postgres.New(
		postgres.Config{
			DSN:                  connection,
			PreferSimpleProtocol: true, // disables implicit prepared statement usage
		},
	), nil
}

func mysqlDialector(connection string) (gorm.Dialector, error) {
	const defaultStringSize = 256

	return mysql.New(
		mysql.Config{
			// e.g. gorm:gorm@tcp(127.0.0.1:3306)/gorm?charset=utf8&parseTime=True&loc=Local
			DSN:                       connection,        // data source name
			DefaultStringSize:         defaultStringSize, // default size for string fields
			DisableDatetimePrecision:  true,              // disable datetime precision, which not supported before MySQL 5.6
			DontSupportRenameIndex:    true,              // drop & create when rename index, rename index not supported before MySQL 5.7, MariaDB
			DontSupportRenameColumn:   true,              // `change` when rename column, rename column not supported before MySQL 8, MariaDB
			SkipInitializeWithVersion: false,             // auto configure based on currently MySQL version
		},
	), nil
}
