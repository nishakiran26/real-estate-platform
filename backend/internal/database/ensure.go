package database

import (
	"database/sql"
	"fmt"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rera/property-platform/backend/internal/config"
)

// safeSchemaName guards the one value that cannot be passed as a bind parameter:
// MySQL does not allow a placeholder for a schema name in CREATE DATABASE.
var safeSchemaName = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

// EnsureDatabase creates the configured schema if it does not exist yet, so a
// local `go run` works against a bare MySQL server without any manual setup.
// Under Docker Compose the schema is already created by the MySQL image and
// this is a no-op.
func EnsureDatabase(cfg *config.Config) error {
	name := cfg.Database.Name
	if !safeSchemaName.MatchString(name) {
		return fmt.Errorf("DB_NAME %q is not a valid schema name", name)
	}

	// Connect to the server itself rather than to the (possibly missing) schema.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql server connection: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("connect to mysql at %s:%s: %w", cfg.Database.Host, cfg.Database.Port, err)
	}

	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name)
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}
